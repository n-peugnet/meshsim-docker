// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"

	"github.com/yaricom/goGraphML/graphml"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

type Node struct {
	id       int64
	IP       netip.Addr
	Hostname string
}

func (n *Node) ID() int64 {
	return n.id
}

func (n *Node) String() string {
	return n.IP.String()
}

const (
	kHostname  = "hostname"
	kIP        = "main-ip-addr"
	kMAC       = "mac-addr"
	kMaxSignal = "max-signal"
	kType      = "type"
)

func ip2id(ip netip.Addr) int64 {
	return int64(binary.BigEndian.Uint32(ip.AsSlice()))
}

func parseNode(attrs map[string]any) (*Node, error) {
	ip, err := netip.ParseAddr(attrs[kIP].(string))
	if err != nil {
		return nil, fmt.Errorf("parse IP: %w", err)
	}
	node := &Node{
		id:       ip2id(ip),
		IP:       ip,
		Hostname: attrs[kHostname].(string),
	}
	return node, nil
}

func parseGraphml(r io.Reader) (graph.Graph, error) {
	document := graphml.NewGraphML("")
	if err := document.Decode(r); err != nil {
		return nil, fmt.Errorf("decode GraphML: %w", err)
	}
	if len(document.Graphs) == 0 {
		return nil, errors.New("no graph in GraphML")
	}
	graphML := document.Graphs[0]

	g := simple.NewUndirectedGraph()
	for _, n := range graphML.Nodes {
		// TODO: store nodes in map to not parse them twice in edges loop
		attrs, _ := n.GetAttributes()
		node, err := parseNode(attrs)
		if err != nil {
			return nil, err
		}
		g.AddNode(node)
	}
	for _, e := range graphML.Edges {
		fromAttrs, _ := e.SourceNode().GetAttributes()
		toAttrs, _ := e.TargetNode().GetAttributes()
		if fromAttrs[kType] == "gate" || toAttrs[kType] == "gate" {
			continue
		}
		from, err := parseNode(fromAttrs)
		if err != nil {
			return nil, fmt.Errorf("source: %w", err)
		}
		to, err := parseNode(toAttrs)
		if err != nil {
			return nil, fmt.Errorf("target: %w", err)
		}
		edge := simple.Edge{F: from, T: to}
		g.SetEdge(edge)
	}
	return g, nil
}
