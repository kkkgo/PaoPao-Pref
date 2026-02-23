package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Resolver struct {
	Name  string
	Stamp string
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
		go func(workerID int) {
			defer wg.Done()
			basePort := 5302 + workerID

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
					status = testResolver(res, basePort)

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
		}(i)
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
	fmt.Println("Testing completed.")

	if data, err := os.ReadFile(banListFile); err == nil {
		sum := sha256.Sum256(data)
		sumLine := hex.EncodeToString(sum[:]) + "  " + filepath.Base(banListFile) + "\n"
		sumFile := banListFile + ".sha256sum"
		if err := os.WriteFile(sumFile, []byte(sumLine), 0644); err != nil {
			fmt.Println("Failed to write sha256sum file:", err)
		} else {
			fmt.Printf("SHA256: %s\nChecksum written to %s\n", strings.TrimSpace(sumLine), sumFile)
		}
	} else {
		fmt.Println("Failed to read ban list for hashing:", err)
	}

	_ = badNew
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

func dnsQuery(host string, qtype string, port int) []string {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", fmt.Sprintf("127.0.0.1:%d", port))
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch strings.ToUpper(qtype) {
	case "MX":
		mxs, err := r.LookupMX(ctx, host)
		if err != nil {
			return nil
		}
		var results []string
		for _, mx := range mxs {
			results = append(results, mx.Host)
		}
		return results
	case "A":
		addrs, err := r.LookupHost(ctx, host)
		if err != nil {
			return nil
		}
		return addrs
	}
	return nil
}

func testResolver(res Resolver, port int) string {
	configPath := fmt.Sprintf("/tmp/test_now_%d.toml", port)
	configContent := fmt.Sprintf(`listen_addresses = ['127.0.0.1:%d']
server_names = ['test']
[static]
  [static.'test']
  stamp = '%s'
`, port, res.Stamp)

	err := ioutil.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		return "CONFIG ERROR"
	}

	cmd := exec.Command("dnscrypt-proxy", "-config", configPath)
	err = cmd.Start()
	if err != nil {
		return "START FAILED"
	}

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	waitForListen(fmt.Sprintf("127.0.0.1:%d", port), time.Second*5)

	var gmailMX []string
	for i := 0; i < 5; i++ {
		gmailMX = dnsQuery("gmail.com", "mx", port)
		if len(gmailMX) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
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

	for _, attempt := range []int{1, 2} {
		_ = attempt
		localIPs := dnsQuery("local.03k.org", "a", port)
		for _, ip := range localIPs {
			if ip == "10.9.8.7" {
				return "OK."
			}
		}
	}

	mxRecords := dnsQuery("03k.org", "mx", port)
	for _, mx := range mxRecords {
		if strings.Contains(strings.ToLower(mx), "qq.com") {
			return "LOCAL BAD."
		}
	}

	return "CONNECT BAD."
}

func waitForListen(addr string, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}