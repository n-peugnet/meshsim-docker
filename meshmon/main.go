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
	monitor := Monitor{
		URL:  os.Getenv("NETGRAPH_URL"),
		Auth: os.Getenv("NETGRAPH_AUTH"),
	}

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
	waker := NewWaker(ip2id(ip))

	monitor.AddHandler(waker)
	monitor.Run(2 * time.Second)
}
