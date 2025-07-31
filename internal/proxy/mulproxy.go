package proxy

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Multi-domain proxy that can handle multiple domains
type MultiDomainProxy struct {
	port    string
	domains map[string]*DomainProxyConfig
}

func NewMultiDomainProxy(port string) *MultiDomainProxy {
	return &MultiDomainProxy{
		port:    port,
		domains: make(map[string]*DomainProxyConfig),
	}
}

func (m *MultiDomainProxy) AddDomain(path string, config *DomainProxyConfig) {
	m.domains[path] = config
}

func (m *MultiDomainProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract the first part of the path to determine target domain
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathParts) == 0 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	domainKey := pathParts[0]
	config, exists := m.domains[domainKey]
	if !exists {
		http.Error(w, fmt.Sprintf("No proxy configured for path: %s", domainKey), http.StatusNotFound)
		return
	}

	log.Printf("Multi-domain proxy: forwarding to %s with original headers: %v", config.TargetDomain, r.Header)

	// Create a temporary proxy for this domain
	proxy := NewDomainProxy(config)
	
	// Modify the request path to remove the domain key
	if len(pathParts) > 1 {
		r.URL.Path = "/" + strings.Join(pathParts[1:], "/")
	} else {
		r.URL.Path = "/"
	}

	proxy.ServeHTTP(w, r)
}

func (m *MultiDomainProxy) Start() error {
	server := &http.Server{
		Addr:         ":" + m.port,
		Handler:      m,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting multi-domain proxy server on port %s", m.port)
	for path, config := range m.domains {
		log.Printf("  /%s/* -> %s://%s", path, config.TargetScheme, config.TargetDomain)
	}
	
	return server.ListenAndServe()
}
