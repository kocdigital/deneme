package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewProxy(rawUrl string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Modify requests
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		
		// Strip /grafana prefix from the path if present
		if strings.HasPrefix(r.URL.Path, "/grafana") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/grafana")
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}
	}

	// Modify response
	proxy.ModifyResponse = func(r *http.Response) error {
		// Add a response header with exact capitalization from the screenshot
		r.Header.Set("X-Debug-Response", "Some Value")
		
		// Only modify HTML responses
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "text/html") {
			// Read the body
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			// Close the original body
			r.Body.Close()
			
			// Replace the appSubUrl empty string with the specified path that includes /grafana
			modifiedBody := bytes.Replace(
				bodyBytes,
				[]byte(`"appSubUrl":""`),
				[]byte(`"appSubUrl":"/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy/grafana"`),
				-1,
			)
			modifiedBody = bytes.Replace(
				modifiedBody,
				[]byte(`"/avatar/`),
				[]byte(`"avatar/`),
				-1,
				)
			
			// Also fix other URLs that might be affected by the subpath
			modifiedBody = bytes.Replace(
				modifiedBody,
				[]byte(`href="/"`),
				[]byte(`href="/grafana/"`),
				-1,
			)
			
			// Update Content-Length header
			r.ContentLength = int64(len(modifiedBody))
			r.Header.Set("Content-Length", string(r.ContentLength))
			
			// Create a new ReadCloser with modified body
			r.Body = io.NopCloser(bytes.NewReader(modifiedBody))
		}
		
		return nil
	}

	return proxy, nil
}

func main() {
	proxy, err := NewProxy("http://grafana-rancher.monitoring/")
	if err != nil {
		panic(err)
	}

	// Handle both root and /grafana path
	http.Handle("/", proxy)
	http.Handle("/grafana/", proxy)
	
	log.Println("Starting reverse proxy server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

