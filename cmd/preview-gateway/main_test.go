//go:build linux

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartingPreviewDoesNotForwardRequests(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request reached backend before readiness")
	}))
	defer backend.Close()
	var ready atomic.Bool
	w := httptest.NewRecorder()
	handler(manifest{BuildID: "new"}, backend.URL, &ready).ServeHTTP(w, httptest.NewRequest("GET", "/login", nil))
	if w.Code != 503 {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestInfoChecksBackendBuildAndHealth(t *testing.T) {
	for _, version := range []string{"preview-new", "preview-old"} {
		t.Run(version, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/n9e/version":
					fmt.Fprint(w, version)
				case "/login":
					fmt.Fprint(w, "<html></html>")
				default:
					fmt.Fprint(w, `{"status":"success"}`)
				}
			}))
			defer backend.Close()
			var ready atomic.Bool
			ready.Store(true)
			w := httptest.NewRecorder()
			handler(manifest{BuildID: "new", RunDir: "/private", CredentialsFile: "/secret"}, backend.URL, &ready).
				ServeHTTP(w, httptest.NewRequest("GET", "/__preview/info", nil))
			var info map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
				t.Fatal(err)
			}
			if info["ready"] != (version == "preview-new") {
				t.Fatalf("wrong readiness: %s", w.Body.String())
			}
			if strings.Contains(w.Body.String(), "/private") || strings.Contains(w.Body.String(), "/secret") {
				t.Fatal("private paths leaked")
			}
		})
	}
}

func TestProxyPreservesAPIQueryAndStreamsBeforeCompletion(t *testing.T) {
	finish := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "q=a%2Bb" || r.Header.Get("Authorization") != "Bearer test" {
			t.Error("request query/auth changed")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-finish
		fmt.Fprint(w, "data: done\n\n")
	}))
	defer backend.Close()
	defer close(finish)
	var ready atomic.Bool
	ready.Store(true)
	gateway := httptest.NewServer(handler(manifest{BuildID: "stream"}, backend.URL, &ready))
	defer gateway.Close()
	request, _ := http.NewRequest("GET", gateway.URL+"/api/chat?q=a%2Bb", nil)
	request.Header.Set("Authorization", "Bearer test")
	streamClient := &http.Client{Timeout: 2 * time.Second}
	response, err := streamClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || line != "data: first\n" {
		t.Fatalf("stream buffered: %q, %v", line, err)
	}
	if response.Header.Get("X-N9E-Preview-Build") != "stream" {
		t.Fatal("missing build identity")
	}
}

func TestBackendFailureIsNotReady(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		io.WriteString(w, "preview-new")
	}))
	defer backend.Close()
	if err := health(backend.URL, manifest{BuildID: "new"}); err == nil {
		t.Fatal("backend HTTP 500 accepted as healthy")
	}
}

func TestLocalOnlyEndpointsNeverForwarded(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("local-only endpoint reached backend")
	}))
	defer backend.Close()
	var ready atomic.Bool
	ready.Store(true)
	for _, route := range []string{"/prometheus/api/v1/write", "/dumper/sync", "/debug/pprof/", "/metrics", "/%70rometheus/api/v1/query", "/x/../prometheus/api/v1/query"} {
		w := httptest.NewRecorder()
		handler(manifest{BuildID: "new"}, backend.URL, &ready).ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d", route, w.Code)
		}
	}
}
