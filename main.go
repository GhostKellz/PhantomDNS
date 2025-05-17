// main.go
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

var (
	serverRunning = false
	serverMutex   sync.Mutex
)

// Config structure for YAML configuration
type Config struct {
	Upstream   string   `yaml:"upstream"`
	Port       string   `yaml:"port"`
	Blocklists []string `yaml:"blocklists"`
	CacheSize  int64    `yaml:"cache_size"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(filePath string) *Config {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return &config
}

// FetchBlocklist fetches and parses a blocklist from a URL
func FetchBlocklist(url string) map[string]struct{} {
	blocklist := make(map[string]struct{})
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch blocklist from %s: %v", url, err)
		return blocklist
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 {
			blocklist[fields[1]] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading blocklist: %v", err)
	}

	return blocklist
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg, upstream string, cache *ristretto.Cache, blocklist map[string]struct{}) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	// Check if the domain is blocked
	domain := r.Question[0].Name
	if _, blocked := blocklist[domain]; blocked {
		log.Printf("Blocked query for domain: %s", domain)
		msg.SetRcode(r, dns.RcodeNameError)
		_ = w.WriteMsg(msg)
		return
	}

	// Check cache
	if cachedResp, found := cache.Get(domain); found {
		log.Printf("Cache hit for domain: %s", domain)
		_ = w.WriteMsg(cachedResp.(*dns.Msg))
		return
	}

	// Query upstream
	c := new(dns.Client)
	resp, _, err := c.Exchange(r, upstream)
	if err != nil {
		log.Printf("[error] upstream query failed: %v", err)
		msg.SetRcode(r, dns.RcodeServerFailure)
	} else {
		msg = resp
		cache.Set(domain, msg, 1)
	}

	_ = w.WriteMsg(msg)
}

func handleDoHRequest(resp *dns.Msg, msg *dns.Msg, upstream string, cache *ristretto.Cache, blocklist map[string]struct{}) {
	// Check if the domain is blocked
	domain := msg.Question[0].Name
	if _, blocked := blocklist[domain]; blocked {
		log.Printf("Blocked query for domain: %s", domain)
		resp.SetRcode(msg, dns.RcodeNameError)
		return
	}

	// Check cache
	if cachedResp, found := cache.Get(domain); found {
		log.Printf("Cache hit for domain: %s", domain)
		*resp = *cachedResp.(*dns.Msg)
		return
	}

	// Query upstream
	c := new(dns.Client)
	upstreamResp, _, err := c.Exchange(msg, upstream)
	if err != nil {
		log.Printf("[error] upstream query failed: %v", err)
		resp.SetRcode(msg, dns.RcodeServerFailure)
	} else {
		*resp = *upstreamResp
		cache.Set(domain, resp, 1)
	}
}

func restartServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverRunning {
		log.Println("Restarting PhantomDNS...")
		os.Exit(0) // Simulate restart by exiting; systemd or a script can restart it.
	} else {
		log.Println("PhantomDNS is not running.")
	}
}

func updateBlocklists(config *Config, blocklist map[string]struct{}) {
	log.Println("Updating blocklists...")
	for _, url := range config.Blocklists {
		for domain := range FetchBlocklist(url) {
			blocklist[domain] = struct{}{}
		}
	}
	log.Println("Blocklists updated.")
}

func showStatus(config *Config, blocklist map[string]struct{}, cache *ristretto.Cache) {
	log.Printf("PhantomDNS Status:\n")
	log.Printf("Listening on: %s\n", config.Port)
	log.Printf("Upstream DNS: %s\n", config.Upstream)
	log.Printf("Blocked domains: %d\n", len(blocklist))
	log.Printf("Cache size: %d\n", config.CacheSize)
}

func startWebUI(blocklist map[string]struct{}, cache *ristretto.Cache) {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>PhantomDNS Status</h1>")
		fmt.Fprintf(w, "<p>Blocked domains: %d</p>", len(blocklist))
		fmt.Fprintf(w, "<p>Cache size: %d</p>", cache.Metrics.CostAdded())
	})

	http.HandleFunc("/blocklist", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<h1>Blocklist</h1>")
		for domain := range blocklist {
			fmt.Fprintf(w, "<p>%s</p>", domain)
		}
	})

	log.Println("Starting Web UI on port 5380...")
	if err := http.ListenAndServe(":5380", nil); err != nil {
		log.Fatalf("Failed to start Web UI: %v", err)
	}
}

func ensureSelfSignedCert(certFile, keyFile string) error {
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return nil // Both files exist
		}
	}
	log.Println("Generating self-signed TLS certificate...")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		Subject:               pkix.Name{CommonName: "localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return err
	}
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}
	return nil
}

func startDoTServer(config *Config, cache *ristretto.Cache, blocklist map[string]struct{}) {
	certFile := "server.crt"
	keyFile := "server.key"
	if err := ensureSelfSignedCert(certFile, keyFile); err != nil {
		log.Fatalf("Failed to generate self-signed certificate: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to load TLS certificate: %v", err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	srv := &dns.Server{
		Addr:      config.Port,
		Net:       "tcp-tls",
		TLSConfig: tlsConfig,
	}

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		handleDNSRequest(w, r, config.Upstream, cache, blocklist)
	})

	log.Println("Starting DoT server on", config.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start DoT server: %v", err)
	}
}

func startDoHServer(config *Config, cache *ristretto.Cache, blocklist map[string]struct{}) {
	http.HandleFunc("/dns-query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		msg := new(dns.Msg)
		if r.Method == "POST" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}
			if err := msg.Unpack(body); err != nil {
				http.Error(w, "Failed to parse DNS message", http.StatusBadRequest)
				return
			}
		} else {
			query := r.URL.Query().Get("dns")
			data, err := base64.RawURLEncoding.DecodeString(query)
			if err != nil {
				http.Error(w, "Failed to decode query", http.StatusBadRequest)
				return
			}
			if err := msg.Unpack(data); err != nil {
				http.Error(w, "Failed to parse DNS message", http.StatusBadRequest)
				return
			}
		}

		resp := new(dns.Msg)
		resp.SetReply(msg)
		resp.Authoritative = true

		// Handle DNS request for DoH
		handleDoHRequest(resp, msg, config.Upstream, cache, blocklist)

		// Write response
		responseData, err := resp.Pack()
		if err != nil {
			http.Error(w, "Failed to pack DNS response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		w.Write(responseData)
	})

	log.Println("Starting DoH server on port 443...")
	if err := http.ListenAndServeTLS(":443", "server.crt", "server.key", nil); err != nil {
		log.Fatalf("Failed to start DoH server: %v", err)
	}
}

func main() {
	// Parse CLI flags
	restart := flag.Bool("r", false, "Restart PhantomDNS")
	update := flag.Bool("u", false, "Update blocklists")
	status := flag.Bool("s", false, "Show status")
	flag.Parse()

	// Load configuration
	config := LoadConfig("config.yaml")

	// Initialize cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10 * config.CacheSize, // 10x cache size for counters
		MaxCost:     config.CacheSize,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}

	// Load blocklists
	blocklist := make(map[string]struct{})
	for _, url := range config.Blocklists {
		for domain := range FetchBlocklist(url) {
			blocklist[domain] = struct{}{}
		}
	}

	// Handle CLI commands
	if *restart {
		restartServer()
		return
	}

	if *update {
		updateBlocklists(config, blocklist)
		return
	}

	if *status {
		showStatus(config, blocklist, cache)
		return
	}

	// Start Web UI in a separate goroutine
	go startWebUI(blocklist, cache)

	// Start DoT server in a separate goroutine
	go startDoTServer(config, cache, blocklist)

	// Start DoH server in a separate goroutine
	go startDoHServer(config, cache, blocklist)

	// Start DNS server
	serverRunning = true
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		handleDNSRequest(w, r, config.Upstream, cache, blocklist)
	})

	srv := &dns.Server{Addr: config.Port, Net: "udp"}
	fmt.Println("PhantomDNS listening on", config.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %s", err.Error())
	}
}
