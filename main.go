package main

import (
	"fmt"
	"net"
	"os"
)

func resolveDNS(domain string) {
	fmt.Printf("Resolving DNS for: %s\n\n", domain)

	ips, err := net.LookupIP(domain)

	if err != nil {
		fmt.Println("Error during DNS looking up: %s\n\n", err)
		return
	}

	fmt.Println("IP List")
	for _, ip := range ips {
		if ip.To4() != nil {
			fmt.Println(" -", ip)
		}
	}

	fmt.Println("\nAAAA Records (IPv6):")
	for _, ip := range ips {
		if ip.To4() == nil {
			fmt.Println(" -", ip)
		}
	}

	// CNAME Records
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		fmt.Printf("\nError looking up CNAME: %v\n", err)
	} else {
		fmt.Printf("\nCNAME Record:\n - %s\n", cname)
	}

	// MX Records
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		fmt.Printf("\nError looking up MX records: %v\n", err)
	} else {
		fmt.Println("\nMX Records (Mail Exchange):")
		for _, mx := range mxRecords {
			fmt.Printf(" - %s (Priority: %d)\n", mx.Host, mx.Pref)
		}
	}

}

func main() {
	fmt.Println("Hello World!\n\n")

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <domain>")
		os.Exit(1)
	}

	domain := os.Args[1]
	resolveDNS(domain)
}
