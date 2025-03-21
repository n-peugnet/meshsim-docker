// SPDX-FileCopyrightText: 2025 Nicolas Peugnet <nicolas.peugnet@lip6.fr>
// SPDX-License-Identifier: GPL-2.0-or-later

package httputils_test

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"gitlab.lip6.fr/ie6/synapse-meshsim/meshmon/httputils"
)

func assertMatch(t *testing.T, expected string, content []byte) {
	ok, _ := regexp.Match(expected, content)
	if !ok {
		t.Errorf("expected %q to match:\n%s", expected, content)
	}
}

func TestLogTransport(t *testing.T) {
	// setup server
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "test")
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// setup client
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	client := &http.Client{Transport: httputils.NewLogTransport(http.DefaultTransport)}

	// 200 response
	_, err := client.Get(ts.URL + "/test")
	if err != nil {
		t.Error("unexpected err: ", err)
	}
	expected := `GET http://.*:[0-9]+/test 200\n$`
	assertMatch(t, expected, buf.Bytes())

	// 404 response
	buf.Reset()
	_, err = client.Get(ts.URL + "/nonexisting")
	if err != nil {
		t.Error("unexpected err: ", err)
	}
	expectations := []string{
		`GET http://.*:[0-9]+/nonexisting 404\n`,
		`request:.*\n`,
		`response:\t404 page not found\n$`,
	}
	for _, expected := range expectations {
		assertMatch(t, expected, buf.Bytes())
	}

	// 405 response
	buf.Reset()
	_, err = client.Post(ts.URL+"/test", "text", bytes.NewBufferString("bonjour"))
	if err != nil {
		t.Error("unexpected err: ", err)
	}
	expectations = []string{
		`POST http://.*:[0-9]+/test 405\n`,
		`request:\tbonjour\n`,
		`response:\tMethod Not Allowed\n$`,
	}
	for _, expected := range expectations {
		assertMatch(t, expected, buf.Bytes())
	}
}
