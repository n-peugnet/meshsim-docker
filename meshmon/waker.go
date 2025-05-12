// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"gitlab.lip6.fr/ie6/synapse-meshsim/meshmon/httputils"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/path"
)

var wakeupClient = http.Client{Transport: httputils.NewLogTransport(http.DefaultTransport)}

type Destination struct {
	*Node
	Path []graph.Node
}

func NewDestination(node graph.Node, path []graph.Node) Destination {
	return Destination{
		Node: node.(*Node),
		Path: path,
	}
}

type DestinationMap struct {
	data  map[int64]bool
	mutex sync.RWMutex
}

func NewDestinationMap() *DestinationMap {
	return &DestinationMap{
		data: make(map[int64]bool),
	}
}

func (m *DestinationMap) Has(dest graph.Node) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.data[dest.ID()]
}

func (m *DestinationMap) Replace(dests []Destination) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	clear(m.data)
	for _, d := range dests {
		m.data[d.ID()] = true
	}
}

type Waker struct {
	id         int64
	knownDests *DestinationMap
}

func NewWaker(id int64) *Waker {
	return &Waker{
		id:         id,
		knownDests: NewDestinationMap(),
	}
}

func (w *Waker) Handle(g graph.Graph) {
	// Find self
	self := g.Node(w.id)
	log.Printf("current: %v", self)

	// Find paths from self
	paths := path.DijkstraFrom(self, g)
	nodes := g.Nodes()
	dests := make([]Destination, 0, nodes.Len())
	log.Printf("nodes count: %d", nodes.Len())
	for nodes.Next() {
		if nodes.Node().ID() == self.ID() {
			continue
		}
		path, weight := paths.To(nodes.Node().ID())
		if len(path) == 0 {
			continue
		}
		dests = append(dests, NewDestination(nodes.Node(), path))
		log.Printf("%v --> %v : %v (weight: %v)", paths.From(), nodes.Node(), path, weight)
	}

	// Find new destinations
	var newDests []Destination
	var newDelayedDests []Destination

	for _, dest := range dests {
		if !w.knownDests.Has(dest) {
			if w.knownDests.Has(dest.Path[1]) {
				log.Printf("delay waking up new dest: %v (%v)", dest.Hostname, dest.IP)
				newDelayedDests = append(newDelayedDests, dest)
			} else {
				newDests = append(newDests, dest)
			}
		}
	}

	w.knownDests.Replace(dests)

	// Wake up new destinations
	for _, dest := range newDests {
		log.Printf("waking up new dest: %v (%v)", dest.Hostname, dest.IP)
		if err := wakeupDestination(dest.Hostname); err != nil {
			log.Print(err)
		}
	}

	time.AfterFunc(3*time.Second, func() { w.wakeupDelayed(newDelayedDests) })
}

func (w *Waker) wakeupDelayed(dests []Destination) {
	for _, d := range dests {
		if !w.knownDests.Has(d) {
			log.Printf("ignoring not reachable delayed dest: %v (%v)", d.Hostname, d.IP)
			continue
		}
		log.Printf("waking up delayed dest: %v (%v)", d.Hostname, d.IP)
		if err := wakeupDestination(d.Hostname); err != nil {
			log.Print(err)
		}
	}
}

func wakeupDestination(hostname string) error {
	url := fmt.Sprintf("http://localhost:8008/_synapse/admin/v1/federation/destinations/%s/reset_connection", hostname)
	request, _ := http.NewRequest("POST", url, nil)
	request.Header.Set("Authorization", "Bearer fake_token")
	response, err := wakeupClient.Do(request)
	if err != nil {
		return err
	}
	response.Body.Close()
	return nil
}
