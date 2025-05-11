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

		// --- Properly handle Set-Cookie headers ---
		setCookies := r.Header["Set-Cookie"]
		newSetCookies := make([]string, 0, len(setCookies))
		for _, cookieHeader := range setCookies {
			// Rewrite Path attribute in each Set-Cookie header
			parts := strings.Split(cookieHeader, ";")
			for i, part := range parts {
				trimmed := strings.TrimSpace(part)
				if strings.HasPrefix(strings.ToLower(trimmed), "path=") {
					// Always set path to /grafana
					parts[i] = " Path=/grafana"
				}
			}
			newSetCookies = append(newSetCookies, strings.Join(parts, ";"))
		}
		if len(newSetCookies) > 0 {
			r.Header["Set-Cookie"] = newSetCookies
		}
		// --- End cookie handling ---

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
				[]byte(`"appSubUrl":"/grafana"`),
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
			
				// Handle client-side cookie manipulation if needed
			modifiedBody = bytes.Replace(
				modifiedBody,
				[]byte(`document.cookie="`),
				[]byte(`document.cookie="path=/grafana; `),
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

	// Add custom transport to handle cookies in redirects if needed
	originalTransport := proxy.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	proxy.Transport = &customTransport{
		originalTransport: originalTransport,
	}

	return proxy, nil
}

// Custom transport to handle cookies in redirects
type customTransport struct {
	originalTransport http.RoundTripper
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log cookies for debugging
	if len(req.Cookies()) > 0 {
		log.Printf("Request cookies: %v", req.Cookies())
	}
	
	resp, err := t.originalTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	
	// Log response cookies for debugging
	if len(resp.Cookies()) > 0 {
		log.Printf("Response cookies: %v", resp.Cookies())
	}
	
	return resp, err
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

