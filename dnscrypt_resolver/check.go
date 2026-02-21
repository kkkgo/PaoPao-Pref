package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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

	for _, res := range resolvers {
		status := testResolver(res)
		if status == "CONNECT BAD." {
			isKnownBadPrefix := false
			for _, badName := range localBadNames {
				if strings.HasPrefix(res.Name, badName+"-") {
					isKnownBadPrefix = true
					break
				}
			}
			if isKnownBadPrefix {
				status = "LOCAL BAD."
			}
		}

		fmt.Printf("%s: %s\n", res.Name, status)

		if status == "LOCAL BAD." {
			localBadNames = append(localBadNames, res.Name)
			badNew = append(badNew, res.Name)
			banMap[res.Name] = true
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

func dig(host string, qtype string) string {
	args := []string{"@127.0.0.1", "-p", "5302", "+time=3", "+tries=1", host, qtype}
	out, _ := exec.Command("dig", args...).CombinedOutput()
	return string(out)
}

func testResolver(res Resolver) string {
	configPath := "/tmp/test_now.toml"
	configContent := fmt.Sprintf(`listen_addresses = ['127.0.0.1:5302']
server_names = ['test']
[static]
  [static.'test']
  stamp = '%s'
`, res.Stamp)

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

	waitForListen("127.0.0.1:5302", time.Second*5)

	var testRes string
	for i := 0; i < 5; i++ {
		testRes = dig("gmail.com", "mx")
		if strings.Contains(testRes, "smtp") || strings.Contains(strings.ToLower(testRes), "google.com") {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if strings.Contains(testRes, "smtp") || strings.Contains(strings.ToLower(testRes), "google.com") {
		if strings.Contains(dig("local.03k.org", "a"), "10.9.8.7") {
			return "OK."
		} else {
			if strings.Contains(dig("local.03k.org", "a"), "10.9.8.7") {
				return "OK."
			} else {
				if strings.Contains(dig("03k.org", "mx"), "qq.com") {
					return "LOCAL BAD."
				} else {
					return "CONNECT BAD."
				}
			}
		}
	} else {
		return "CONNECT BAD."
	}
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
