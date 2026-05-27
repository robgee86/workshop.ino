// Command workshopify serves a Markdown-authored workshop lab handbook over the
// local network. Content lives on disk and is read per request, so instructors
// can fix typos mid-workshop without rebuilding or restarting.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"workshopify/internal/server"
)

func main() {
	contentDir := flag.String("content", "./content", "path to the workshop content directory")
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	root, err := filepath.Abs(*contentDir)
	if err != nil {
		log.Fatalf("resolving content path: %v", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		log.Fatalf("content directory not found: %s", root)
	}

	handler, err := server.New(root)
	if err != nil {
		log.Fatalf("starting server: %v", err)
	}

	printURLs(*addr, root)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}

// printURLs lists the addresses participants can use to reach the handbook,
// including LAN IPs so they can connect from their own laptops.
func printURLs(addr, root string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "8080"
	}
	fmt.Printf("\n  workshopify  ·  serving %s\n\n", root)
	fmt.Printf("    http://localhost:%s\n", port)
	for _, ip := range lanIPv4s() {
		fmt.Printf("    http://%s:%s\n", ip, port)
	}
	fmt.Println()
}

func lanIPv4s() []string {
	var ips []string
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			ips = append(ips, ip4.String())
		}
	}
	return ips
}
