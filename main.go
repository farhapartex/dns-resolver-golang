package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/idna"
)

var dnsCache = make(map[string]struct {
	records map[string][]string
	expiry  time.Time
})
var cacheMutex sync.Mutex

func initLogger() {
	file, err := os.OpenFile("dns_resolver.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	log.SetOutput(file)
}

// normalizeDomain handles internationalized domain names (IDNs)
func normalizeDomain(domain string) string {
	normalized, err := idna.ToASCII(domain)
	if err != nil {
		log.Printf("Error normalizing domain: %v\n", err)
		return domain
	}
	return normalized
}

// cacheResult caches DNS records for a domain
func cacheResult(domain string, records map[string][]string) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	dnsCache[domain] = struct {
		records map[string][]string
		expiry  time.Time
	}{records: records, expiry: time.Now().Add(10 * time.Minute)}
}

// getCachedResult retrieves cached DNS records
func getCachedResult(domain string) (map[string][]string, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	entry, exists := dnsCache[domain]
	if !exists || time.Now().After(entry.expiry) {
		return nil, false
	}
	return entry.records, true
}

// resolveDNS resolves DNS for a single domain
func resolveDNS(domain string) map[string][]string {
	domain = normalizeDomain(domain)
	if cached, found := getCachedResult(domain); found {
		fmt.Println("Cache hit!")
		return cached
	}

	records := make(map[string][]string)

	// A and AAAA Records
	ips, err := net.LookupIP(domain)
	if err == nil {
		for _, ip := range ips {
			if ip.To4() != nil {
				records["A"] = append(records["A"], ip.String())
			} else {
				records["AAAA"] = append(records["AAAA"], ip.String())
			}
		}
	} else {
		log.Printf("Error looking up IP: %v\n", err)
	}

	// CNAME Record
	cname, err := net.LookupCNAME(domain)
	if err == nil {
		records["CNAME"] = []string{cname}
	} else {
		log.Printf("Error looking up CNAME: %v\n", err)
	}

	// MX Records
	mxRecords, err := net.LookupMX(domain)
	if err == nil {
		for _, mx := range mxRecords {
			records["MX"] = append(records["MX"], fmt.Sprintf("%s (Priority: %d)", mx.Host, mx.Pref))
		}
	} else {
		log.Printf("Error looking up MX: %v\n", err)
	}

	// TXT Records
	txtRecords, err := net.LookupTXT(domain)
	if err == nil {
		records["TXT"] = txtRecords
	} else {
		log.Printf("Error looking up TXT: %v\n", err)
	}

	// NS Records
	nsRecords, err := net.LookupNS(domain)
	if err == nil {
		for _, ns := range nsRecords {
			records["NS"] = append(records["NS"], ns.Host)
		}
	} else {
		log.Printf("Error looking up NS: %v\n", err)
	}

	cacheResult(domain, records)
	return records
}

// reverseDNS performs reverse DNS lookup for an IP
func reverseDNS(ip string) {
	hosts, err := net.LookupAddr(ip)
	if err != nil {
		log.Printf("Error during reverse DNS lookup: %v\n", err)
		return
	}
	fmt.Println("Reverse DNS:")
	for _, host := range hosts {
		fmt.Println(" -", host)
	}
}

// resolveBatch reads domains from a file and resolves them
func resolveBatch(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error reading file: %v\n", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var wg sync.WaitGroup
	for scanner.Scan() {
		domain := scanner.Text()
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()
			fmt.Printf("\nResolving: %s\n", domain)
			records := resolveDNS(domain)
			printRecords(domain, records)
		}(domain)
	}
	wg.Wait()
}

// printRecords formats and prints DNS records
func printRecords(domain string, records map[string][]string) {
	fmt.Printf("\nDNS Records for %s:\n", domain)
	for recordType, values := range records {
		fmt.Printf("%s Records:\n", recordType)
		for _, value := range values {
			fmt.Println(" -", value)
		}
	}
}

func main() {
	initLogger()

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <domain|ip> [--reverse|--file <path>|--server <dns_server>]")
		return
	}

	arg := os.Args[1]
	switch {
	case strings.Contains(arg, "."):
		if len(os.Args) == 4 && os.Args[2] == "--server" {
			fmt.Println("Coming soon ...")
		} else {
			records := resolveDNS(arg)
			printRecords(arg, records)
		}
	case strings.Contains(arg, ":"):
		reverseDNS(arg)
	case len(os.Args) == 3 && os.Args[2] == "--file":
		resolveBatch(arg)
	default:
		fmt.Println("Invalid input.")
	}
}
