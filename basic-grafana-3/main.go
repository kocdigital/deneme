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
		// The X-Debug-Request header was not showing in the screenshot
		// So we'll leave it out for now
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
			
			// Replace the appSubUrl empty string with the specified path
			modifiedBody := bytes.Replace(
				bodyBytes,
				[]byte(`"appSubUrl":""`),
				[]byte(`"appSubUrl":"/k8s/clusters/c-m-8dbg2lrc/api/v1/namespaces/default/services/http:golang:8080/proxy"`),
				-1,
			)
			modifiedBody = bytes.Replace(
				modifiedBody,
				[]byte(`"/avatar/`),
				[]byte(`"avatar/`),
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

	http.Handle("/", proxy)
	log.Println("Starting reverse proxy server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

