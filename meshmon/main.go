// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"log"
	"net"
	"net/netip"
	"os"
	"slices"
	"time"

	"gonum.org/v1/gonum/graph/path"
)

func main() {
	url := os.Getenv("NETGRAPH_URL")
	auth := os.Getenv("NETGRAPH_AUTH")
	var ip netip.Addr
	iface, _ := net.InterfaceByName("eth0")
	addrs, _ := iface.Addrs()
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip, ok = netip.AddrFromSlice(ipnet.IP.To4()); ok {
				break
			}
		}
	}
	log.Println("ip:", ip)
	id := ip2id(ip)

	client, err := createClient(url)
	if err != nil {
		panic(err)
	}
	request, err := createRequest(url, auth)
	if err != nil {
		panic(err)
	}

	var prevDests []*Node
	c := time.Tick(2 * time.Second)
	for range c {
		// Obtain graph
		g, err := requestGraph(client, request)
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}

		// Find self
		self := g.Node(id)
		log.Printf("current: %v", self)

		// Find paths from self
		paths := path.DijkstraFrom(self, g)
		nodes := g.Nodes()
		dests := make([]*Node, 0, nodes.Len())
		log.Printf("nodes count: %d", nodes.Len())
		for nodes.Next() {
			if nodes.Node().ID() == self.ID() {
				continue
			}
			path, weight := paths.To(nodes.Node().ID())
			if len(path) == 0 {
				continue
			}
			dests = append(dests, nodes.Node().(*Node))
			log.Printf("%v --> %v : %v (weight: %v)", paths.From(), nodes.Node(), path, weight)
		}

		// Wakeup new destinations
		for _, dest := range dests {
			idx := slices.IndexFunc(prevDests, func(n *Node) bool {
				return n.id == dest.id
			})
			if idx == -1 {
				log.Printf("waking up to dest: %v", dest.Hostname)
				if err := wakeupDestination(dest.Hostname); err != nil {
					log.Print(err)
				}
			}
		}

		// Save dests
		prevDests = dests
	}
}
