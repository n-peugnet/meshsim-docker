// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"gitlab.lip6.fr/ie6/synapse-meshsim/meshmon/httputils"
	"gonum.org/v1/gonum/graph"
)

type Handler interface {
	Handle(graph.Graph)
}

type Monitor struct {
	URL      string
	Auth     string
	Handlers []Handler
}

func (m *Monitor) AddHandler(handler Handler) {
	m.Handlers = append(m.Handlers, handler)
}

func (m *Monitor) Run(interval time.Duration) {
	client, err := createClient(m.URL)
	if err != nil {
		panic(err)
	}
	request, err := createRequest(m.URL, m.Auth)
	if err != nil {
		panic(err)
	}

	c := time.Tick(interval)
	for range c {
		// Obtain graph
		g, err := requestGraph(client, request)
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}

		for _, handler := range m.Handlers {
			handler.Handle(g)
		}
	}
}

func createClient(netgraphURLStr string) (*http.Client, error) {
	netgraphURL, err := url.Parse(netgraphURLStr)
	if err != nil {
		return nil, fmt.Errorf("incorrect netgraph URL: %w", err)
	}

	var transport http.RoundTripper
	switch netgraphURL.Scheme {
	case "file":
		// Allow to use file:// URLs for debugging purposes
		t := &http.Transport{}
		t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
		transport = t
	case "https":
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	default:
		transport = http.DefaultTransport
	}
	return &http.Client{Transport: httputils.NewLogTransport(transport)}, nil
}

func createRequest(netgraphURLStr string, auth string) (*http.Request, error) {
	request, err := http.NewRequest("GET", netgraphURLStr, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	request.Header.Set("Authorization", "Basic "+auth)
	request.Header.Set("Accept", "application/graphml+xml")
	return request, nil
}

func requestGraph(client *http.Client, request *http.Request) (graph.Graph, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("do request '%s': %v", request.URL.String(), err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("received invalid status: %s", response.Status)
	}
	g, err := parseGraphml(response.Body)
	if err != nil {
		return nil, fmt.Errorf("parse graphml: %w", err)
	}
	return g, nil
}
