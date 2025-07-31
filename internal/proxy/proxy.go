package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type DomainProxyConfig struct {
	ProxyPort    string // Port where Server 2 will listen for Server 3
	TargetDomain string // Target domain (e.g., "api.example.com")
	TargetScheme string // "http" or "https"
	PreservePath bool   // Whether to preserve the full path or replace it
}

type DomainProxy struct {
	config       *DomainProxyConfig
	reverseProxy *httputil.ReverseProxy
}

func NewDomainProxy(config *DomainProxyConfig) *DomainProxy {
	// Parse target URL
	targetURL := &url.URL{
		Scheme: config.TargetScheme,
		Host:   config.TargetDomain,
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize transport
	proxy.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Set to true only if needed for self-signed certs
		},
	}

	// Custom director to modify requests
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Log the proxied request
		log.Printf("Proxying: %s %s -> %s://%s%s",
			req.Method, req.URL.Path, config.TargetScheme, config.TargetDomain, req.URL.Path)

		// Only set the Host header for proper domain routing
		// This is required for the request to reach the correct domain
		req.Host = config.TargetDomain
		req.Header.Set("Host", config.TargetDomain)

		// DO NOT add any other headers - preserve client's original headers
		// All other headers (Authorization, User-Agent, Content-Type, etc.)
		// are passed through unchanged from the client
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Real-IP")

		log.Printf("Forwarding headers from client: %v", req.Header)
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error for %s %s: %v", r.Method, r.URL.Path, err)

		if strings.Contains(err.Error(), "no such host") {
			http.Error(w, fmt.Sprintf("Domain %s cannot be resolved", config.TargetDomain), http.StatusBadGateway)
		} else if strings.Contains(err.Error(), "connection refused") {
			http.Error(w, fmt.Sprintf("Domain %s is unreachable", config.TargetDomain), http.StatusBadGateway)
		} else if strings.Contains(err.Error(), "timeout") {
			http.Error(w, fmt.Sprintf("Request to %s timed out", config.TargetDomain), http.StatusGatewayTimeout)
		} else {
			http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		}
	}

	return &DomainProxy{
		config:       config,
		reverseProxy: proxy,
	}
}

func (p *DomainProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	log.Printf("Received request from %s: %s %s", getClientIP(r), r.Method, r.URL.Path)
	log.Printf("Client headers: %v", r.Header)

	// DO NOT add CORS headers automatically - let the client control this
	// If the target domain sends CORS headers, they will be passed through
	// If you need CORS, send them from your client (Server 3)

	// Handle preflight requests only if the client sends them
	if r.Method == "OPTIONS" {
		// Just forward the OPTIONS request to the target domain
		// Don't handle it here - let the target domain respond
	}

	// Forward the request to the target domain with original headers
	p.reverseProxy.ServeHTTP(w, r)
}

func (p *DomainProxy) Start() error {
	server := &http.Server{
		Addr:         ":" + p.config.ProxyPort,
		Handler:      p,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting domain proxy server on port %s", p.config.ProxyPort)
	log.Printf("Forwarding requests to %s://%s", p.config.TargetScheme, p.config.TargetDomain)
	log.Printf("Access the domain via: http://your-server2-ip:%s", p.config.ProxyPort)

	return server.ListenAndServe()
}

// Utility functions
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
