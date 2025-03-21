// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas.peugnet@lip6.fr>
// SPDX-License-Identifier: GPL-2.0-or-later

package httputils

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

// LogTransport is a [http.RoundTripper] that logs each request with its
// associated response.
type LogTransport struct {
	t   http.RoundTripper
	log *log.Logger
}

// NewLogTransport creates a new [LogTransport] using the default Logger
// and the default Transport.
func NewLogTransport(t http.RoundTripper) *LogTransport {
	return &LogTransport{
		t,
		log.Default(),
	}
}

// RoundTrip implements the [http.RoundTripper] interface, by calling the
// underlying RoundTripper.
func (lt *LogTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	reqBody := []byte("")
	if req.Body != nil {
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			panic(err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
	}
	res, err = lt.t.RoundTrip(req)
	if err != nil {
		return
	}
	if res.StatusCode >= 300 {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return res, err
		}
		res.Body.Close()
		res.Body = io.NopCloser(bytes.NewReader(resBody))
		lt.log.Printf(
			"%s %s %d\nrequest:\t%s\nresponse:\t%s",
			req.Method, req.URL, res.StatusCode, reqBody, resBody,
		)
	} else {
		lt.log.Printf("%s %s %d", req.Method, req.URL, res.StatusCode)
	}
	return
}
