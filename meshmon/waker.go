// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"fmt"
	"log"
	"net/http"
	"slices"

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

type Waker struct {
	id           int64
	prevDests    map[int64]bool
	delayedDests []*Node
}

func NewWaker(id int64) *Waker {
	return &Waker{
		id:        id,
		prevDests: make(map[int64]bool),
	}
}

func (a *Waker) Handle(g graph.Graph) {
	// Find self
	self := g.Node(a.id)
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

	// Wakeup delayed destinations
	for _, n := range a.delayedDests {
		reachable := slices.ContainsFunc(dests, func(d Destination) bool {
			return d.ID() == n.ID()
		})
		if !reachable {
			log.Printf("ignoring not reachable delayed dest: %v", n.Hostname)
			continue
		}
		log.Printf("waking up delayed dest: %v", n.Hostname)
		if err := wakeupDestination(n.Hostname); err != nil {
			log.Print(err)
		}
	}
	a.delayedDests = nil

	// Wakeup new destinations
alldests:
	for _, dest := range dests {
		if !a.prevDests[dest.ID()] {
			for _, n := range dest.Path[1:] {
				if a.prevDests[n.ID()] {
					log.Printf("delay waking up new dest: %v", dest.Hostname)
					a.delayedDests = append(a.delayedDests, dest.Node)
					continue alldests
				}
			}
			log.Printf("waking up new dest: %v", dest.Hostname)
			if err := wakeupDestination(dest.Hostname); err != nil {
				log.Print(err)
			}
		}
	}

	// Save dests
	clear(a.prevDests)
	for _, d := range dests {
		a.prevDests[d.ID()] = true
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
