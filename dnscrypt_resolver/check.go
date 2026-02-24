package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
)

type Resolver struct {
	Name  string
	Stamp string
}


func queryUpstream(stamp, host string, qtype uint16) []dns.RR {
	opts := &upstream.Options{
		Timeout: 10 * time.Second,
	}
	u, err := upstream.AddressToUpstream(stamp, opts)
	if err != nil {
		return nil
	}

	req := &dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{{
		Name:   dns.Fqdn(host),
		Qtype:  qtype,
		Qclass: dns.ClassINET,
	}}

	reply, err := u.Exchange(req)
	if err != nil || reply == nil {
		return nil
	}
	return reply.Answer
}

func lookupA(stamp, host string) []string {
	ans := queryUpstream(stamp, host, dns.TypeA)
	var ips []string
	for _, rr := range ans {
		if a, ok := rr.(*dns.A); ok {
			ips = append(ips, a.A.String())
		}
	}
	return ips
}

func lookupMX(stamp, host string) []string {
	ans := queryUpstream(stamp, host, dns.TypeMX)
	var hosts []string
	for _, rr := range ans {
		if mx, ok := rr.(*dns.MX); ok {
			hosts = append(hosts, strings.TrimSuffix(mx.Mx, "."))
		}
	}
	return hosts
}

func testResolver(res Resolver) string {
	var gmailMX []string
	for i := 0; i < 3; i++ {
		gmailMX = lookupMX(res.Stamp, "gmail.com")
		if len(gmailMX) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	gmailOk := false
	for _, mx := range gmailMX {
		mx = strings.ToLower(mx)
		if strings.Contains(mx, "smtp") || strings.Contains(mx, "google.com") {
			gmailOk = true
			break
		}
	}
	if !gmailOk {
		return "CONNECT BAD."
	}

	localIPs := lookupA(res.Stamp, "local.03k.org")
	for _, ip := range localIPs {
		if ip == "10.9.8.7" {
			return "OK."
		}
	}

	nipOk := false
	nip1 := lookupA(res.Stamp, "10.0.0.1.nip.io")
	for _, ip := range nip1 {
		if ip == "10.0.0.1" {
			nipOk = true
			break
		}
	}
	if nipOk {
		nipOk = false
		nip2 := lookupA(res.Stamp, "192-168-1-250.nip.io")
		for _, ip := range nip2 {
			if ip == "192.168.1.250" {
				nipOk = true
				break
			}
		}
	}
	if nipOk {
		return "OK."
	}

	mxRecords := lookupMX(res.Stamp, "03k.org")
	for _, mx := range mxRecords {
		if strings.Contains(strings.ToLower(mx), "qq.com") {
			return "LOCAL BAD."
		}
	}

	return "CONNECT BAD."
}

func main() {
	fmt.Println("Downloading public-resolvers.md...")
	resolvers := fetchResolvers()
	if len(resolvers) == 0 {
		fmt.Println("No resolvers found!")
		os.Exit(1)
	}
	fmt.Printf("Found %d resolvers.\n", len(resolvers))

	for _, r := range resolvers {
		fmt.Println(r.Name)
	}

	banListFile := "/data/ban_list.txt"
	banMap := make(map[string]bool)
	if data, err := ioutil.ReadFile(banListFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				banMap[line] = true
			}
		}
	}

	ips, err := net.LookupHost("local.03k.org")
	localOk := false
	if err == nil {
		for _, ip := range ips {
			if ip == "10.9.8.7" {
				localOk = true
				break
			}
		}
	}
	if !localOk {
		fmt.Printf("Test record failed on local network\n")
		os.Exit(1)
	}
	fmt.Println("Ready to test...")

	var badNew []string
	var localBadNames []string
	var mu sync.Mutex

	numWorkers := 15
	jobs := make(chan Resolver, len(resolvers))
	type result struct {
		Name   string
		Status string
	}
	results := make(chan result)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for res := range jobs {
				mu.Lock()
				isKnownBadPrefix := false
				for _, badName := range localBadNames {
					if strings.HasPrefix(res.Name, badName+"-") {
						isKnownBadPrefix = true
						break
					}
				}
				mu.Unlock()

				var status string
				if isKnownBadPrefix {
					status = "LOCAL BAD."
				} else {
					status = testResolver(res)

					if status == "CONNECT BAD." {
						mu.Lock()
						for _, badName := range localBadNames {
							if strings.HasPrefix(res.Name, badName+"-") {
								status = "LOCAL BAD."
								break
							}
						}
						mu.Unlock()
					}
				}

				if status == "LOCAL BAD." {
					mu.Lock()
					localBadNames = append(localBadNames, res.Name)
					mu.Unlock()
				}

				results <- result{res.Name, status}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, res := range resolvers {
			jobs <- res
		}
		close(jobs)
	}()

	for r := range results {
		fmt.Printf("%s: %s\n", r.Name, r.Status)
		if r.Status == "LOCAL BAD." {
			badNew = append(badNew, r.Name)
			banMap[r.Name] = true
		}
	}

	var allBans []string
	for k := range banMap {
		allBans = append(allBans, k)
	}
	sort.Strings(allBans)

	os.MkdirAll(filepath.Dir(banListFile), 0755)
	if f, err := os.OpenFile(banListFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
		for _, b := range allBans {
			f.WriteString(b + "\n")
		}
		f.Close()
	} else {
		fmt.Println("Failed to write to ban list:", err)
	}

	_ = badNew
	fmt.Println("Testing completed.")
}

func fetchResolvers() []Resolver {
	sources := []struct {
		url    string
		prefix string
	}{
		{"https://raw.githubusercontent.com/DNSCrypt/dnscrypt-resolvers/master/v3/public-resolvers.md", ""},
		{"https://www.dnscry.pt/resolvers.md", ""},
	}

	var resolvers []Resolver
	namesFound := make(map[string]bool)

	for _, src := range sources {
		resp, err := http.Get(src.url)
		if err != nil {
			fmt.Printf("Failed to download from %s: %v\n", src.url, err)
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		var currentName string

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "## ") {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) == 2 {
					currentName = src.prefix + strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "sdns://") && currentName != "" {
				if !namesFound[currentName] {
					resolvers = append(resolvers, Resolver{Name: currentName, Stamp: line})
					namesFound[currentName] = true
				}
			}
		}
		resp.Body.Close()
	}

	sort.Slice(resolvers, func(i, j int) bool {
		return resolvers[i].Name < resolvers[j].Name
	})

	return resolvers
}
