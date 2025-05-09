// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"log"
	"net"
	"net/netip"
	"os"
	"time"
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

	client, err := createClient(url)
	if err != nil {
		panic(err)
	}
	request, err := createRequest(url, auth)
	if err != nil {
		panic(err)
	}

	waker := NewWaker(ip2id(ip))
	c := time.Tick(2 * time.Second)
	for range c {
		// Obtain graph
		g, err := requestGraph(client, request)
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}

		waker.WakeNewDestinations(g)
	}
}
