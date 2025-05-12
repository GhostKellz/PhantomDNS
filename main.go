// main.go
package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dgraph-io/ristretto"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
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

func main() {
	// Load configuration
	config := LoadConfig("config.yaml")

	if os.Geteuid() != 0 {
		log.Fatal("PhantomDNS must be run as root to bind to port 53")
	}

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

	// Use the upstream server from the config
	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		handleDNSRequest(w, r, config.Upstream, cache, blocklist)
	})

	srv := &dns.Server{Addr: config.Port, Net: "udp"}
	fmt.Println("PhantomDNS listening on", config.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %s", err.Error())
	}
}
