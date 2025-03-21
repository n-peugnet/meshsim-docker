// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>

package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"gitlab.lip6.fr/ie6/synapse-meshsim/meshmon/httputils"
	"gonum.org/v1/gonum/graph"
)

var wakeupClient = http.Client{Transport: httputils.NewLogTransport(http.DefaultTransport)}

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
