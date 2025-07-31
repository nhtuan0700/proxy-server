package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/htuan0700/proxy-server/internal/configs"
	"github.com/htuan0700/proxy-server/internal/proxy"
)

// Health check for domain
func domainHealthCheck(domain, scheme string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
		}

		healthURL := fmt.Sprintf("%s://%s/", scheme, domain)
		resp, err := client.Get(healthURL)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Domain %s is unreachable: %v", domain, err)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Domain %s is healthy (Status: %d)", domain, resp.StatusCode)
	}
}

func main() {
	// Choose one of the following modes:
	/*
		// MODE 1: Single Domain Proxy
		log.Println("=== Single Domain Proxy Mode ===")
		config := &DomainProxyConfig{
			ProxyPort:    "8081",           // Port on Server 2
			TargetDomain: "example.com", // Target domain
			TargetScheme: "https",          // http or https
			PreservePath: true,             // Keep original paths
		}

		proxy := NewDomainProxy(config)

		// Add health check
		http.HandleFunc("/health", domainHealthCheck(config.TargetDomain, config.TargetScheme))

		// Handle all requests through proxy
		http.Handle("/", proxy)

		if err := proxy.Start(); err != nil {
			log.Fatal("Failed to start proxy server:", err)
		}
	*/

	// MODE 2: Multi-Domain Proxy (uncomment to use)
	log.Println("=== Multi-Domain Proxy Mode ===")

	multiProxy := proxy.NewMultiDomainProxy("8000")

	cfg, err := configs.NewConfig("/app/configs/config.yml")
	if err != nil {
		panic(err)
	}

	for _, target := range cfg.TargetDomains {
		multiProxy.AddDomain(target.Key, &proxy.DomainProxyConfig{
			TargetDomain: target.Domain,
			TargetScheme: target.Schema,
			PreservePath: true,
		})
	}

	if err := multiProxy.Start(); err != nil {
		log.Fatal("Failed to start multi-domain proxy:", err)
	}
}

/*
HEADER PASSTHROUGH BEHAVIOR:

✅ PRESERVED FROM CLIENT (Server 3):
- Authorization: Bearer token123
- Content-Type: application/json
- User-Agent: MyApp/1.0
- X-API-Key: secret123
- Custom-Header: custom-value
- Accept: application/json
- Cookie: session=abc123
- All other custom headers

❌ ONLY MODIFIED BY PROXY:
- Host: target-domain.com (required for proper routing)

🔄 RESPONSE HEADERS:
- All response headers from target domain are passed back unchanged
- No CORS headers added by proxy
- No modification of response headers

USAGE EXAMPLES:

1. CLIENT REQUEST (Server 3 → Server 2):
   curl -H "Authorization: Bearer abc123" \
        -H "Content-Type: application/json" \
        -H "User-Agent: MyApp/1.0" \
        http://server2:8080/api/users

2. FORWARDED REQUEST (Server 2 → Target Domain):
   GET /api/users HTTP/1.1
   Host: api.example.com              ← Only this is changed
   Authorization: Bearer abc123       ← Preserved
   Content-Type: application/json     ← Preserved
   User-Agent: MyApp/1.0             ← Preserved

3. If you need additional headers, send them from your client:
   curl -H "X-Forwarded-For: 192.168.1.100" \
        -H "X-Real-IP: 192.168.1.100" \
        http://server2:8080/api/users
*/
