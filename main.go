// Command workshop.ino serves a Markdown-authored workshop lab handbook. It runs
// on the participant's own device: the handbook is opened in a browser there,
// and applying a step's "solution" writes to the Arduino app living on that same
// device. Content is read from disk per request, so authors can fix typos
// mid-workshop without rebuilding or restarting.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"workshop.ino/internal/server"
)

func main() {
	contentDir := flag.String("content", "./content", "path to the workshop content directory")
	addr := flag.String("addr", ":8080", "address to listen on")
	appsDir := flag.String("apps", defaultAppsDir(), "directory holding the Arduino apps a step's solution is applied to (default ~/ArduinoApps)")
	flag.Parse()

	root, err := filepath.Abs(*contentDir)
	if err != nil {
		log.Fatalf("resolving content path: %v", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		log.Fatalf("content directory not found: %s", root)
	}

	handler, err := server.New(root, *appsDir)
	if err != nil {
		log.Fatalf("starting server: %v", err)
	}

	printURLs(*addr, root)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}

// defaultAppsDir is ~/ArduinoApps when a home directory can be resolved, falling
// back to /home/arduino/ArduinoApps otherwise. This fallback keeps the device
// and the Docker image behaving the same.
func defaultAppsDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "ArduinoApps")
	}
	return "/home/arduino/ArduinoApps"
}

// printURLs lists the addresses the handbook is reachable at: localhost on the
// device itself, plus any LAN IPs (handy for reaching it while authoring).
func printURLs(addr, root string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "8080"
	}
	fmt.Printf("\n  workshop.ino  ·  serving %s\n\n", root)
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
