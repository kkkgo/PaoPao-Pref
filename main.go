package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	file     string
	limit    int
	line     int
	count    int64
	succ     int64
	total    int64
	wg       sync.WaitGroup
	help     bool
	verbose  bool
	server   string
	port     int
	output   bool
	timeout  time.Duration
	sleep    time.Duration
	resolver *net.Resolver
)

func init() {
	flag.StringVar(&file, "file", "domains.txt", "text file containing domain names")
	flag.IntVar(&limit, "limit", 10, "concurrency limit")
	flag.IntVar(&line, "line", 0, "start line")
	flag.BoolVar(&verbose, "v", false, "output nslookup results")
	flag.BoolVar(&help, "h", false, "show help information")
	flag.StringVar(&server, "server", "", "DNS server to use")
	flag.DurationVar(&timeout, "timeout", time.Second*5, "Query timeout")
	flag.DurationVar(&sleep, "sleep", time.Millisecond*1500, "Query sleep")
	flag.IntVar(&port, "port", 53, "DNS port")
	flag.Parse()
}

func main() {
	if help {
		flag.Usage()
		os.Exit(0)
	}
	if server == "" {
		server = os.Getenv("DNS_SERVER")
	}
	if envPort, ok := os.LookupEnv("DNS_PORT"); ok {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}
	if envLine, ok := os.LookupEnv("DNS_LINE"); ok {
		if li, err := strconv.Atoi(envLine); err == nil {
			line = li
		}
	}
	if envLimit, ok := os.LookupEnv("DNS_LIMIT"); ok {
		if lim, err := strconv.Atoi(envLimit); err == nil {
			limit = lim
		}
	}
	if envTimeout, ok := os.LookupEnv("DNS_TIMEOUT"); ok {
		if t, err := time.ParseDuration(envTimeout); err == nil {
			timeout = t
		}
	}
	if envSleep, ok := os.LookupEnv("DNS_SLEEP"); ok {
		if s, err := time.ParseDuration(envSleep); err == nil {
			sleep = s
		}
	}
	if os.Getenv("FILE_OUTPUT") == "yes" {
		output = true
	}
	if os.Getenv("DNS_LOG") == "yes" {
		verbose = true
	}
	if server != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, network, net.JoinHostPort(server, strconv.Itoa(port)))
			},
		}
	} else {
		fmt.Printf("\n\033[33m%s\033[0m\n", "You must specify a DNS server: -server ...")
		if output {
			resolver = net.DefaultResolver
		} else {
			os.Exit(1)
		}
	}
	start := time.Now()
	var elapsed float64
	var average float64
	var estimate float64

	f, err := os.Open(file)
	if err != nil {
		flag.Usage()
		fmt.Println(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		total++
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Total Line :", total)
	fmt.Println("Concurrency Limit:", limit)
	fmt.Println("Timeout:", timeout)
	fmt.Println("Sleep:", sleep)
	var dnsserver string
	if server == "" {
		dnsserver = "system default"
	} else {
		dnsserver = server
	}
	fmt.Println("DNS Server:", dnsserver)
	fmt.Println("DNS Port:", port)
	f.Seek(0, 0)
	if line > 0 {
		fmt.Println("Test :", line, "~", total)
		total = total - int64(line)
		fmt.Println(total, "lines.")
		if total <= 0 {
			fmt.Println("Error line.")
			os.Exit(0)
		}
	}
	ch := make(chan string, limit)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(f)
		newline := 0
		for scanner.Scan() {
			if line > newline {
				newline++
				continue
			}
			domain := scanner.Text()
			ch <- domain
			fmt.Printf("\rNslookup %d/%d domains[%.4f%%]. Succ rate:%.2f%%. Avg time: %.6f seconds. Est time: %s.", count, total, 100*float64(count)/float64(total), 100*float64(succ)/float64(count), average, formatDuration(estimate))
		}
		if err := scanner.Err(); err != nil {
			fmt.Println(err)
		}
		close(ch)
	}()
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range ch {
				now := time.Now()
				time.Sleep(sleep)
				if nslookup(domain) {
					atomic.AddInt64(&succ, 1)
				}
				atomic.AddInt64(&count, 1)
				elapsed = now.Sub(start).Seconds()
				if count > 0 {
					average = elapsed / float64(count)
					estimate = float64(total-count) * average
				}
				fmt.Printf("\rNslookup %d/%d domains[%.4f%%]. Succ rate:%.2f%%. Avg time: %.6f seconds. Est time: %s.", count, total, 100*float64(count)/float64(total), 100*float64(succ)/float64(count), average, formatDuration(estimate))
			}
		}()
	}
	select {
	case <-sig:
		end := time.Now()
		duration := end.Sub(start).Seconds()
		fmt.Printf("\nInterrupted. Processed %d/%d domains in %.2f seconds.\n", count, total, duration)
		os.Exit(0)
	case <-wait(&wg):
		end := time.Now()
		duration := end.Sub(start).Seconds()
		fmt.Printf("\nDone. Processed %d/%d domains in %.2f seconds.\n", count, total, duration)
	}
}

func nslookup(domain string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	r, err := resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		if verbose {
			fmt.Printf("\n\033[33m%s\033[0m\n", err)
		}
		return false
	}
	if verbose {
		fmt.Printf("\n\033[31m%s\033[0m\033[32m%s\033[0m\n", domain, r)
	}
	if output {
		appendToFile(domain)
	}
	return true

}

func wait(wg *sync.WaitGroup) chan struct{} {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		ch <- struct{}{}
	}()
	return ch
}
func formatDuration(seconds float64) string {
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours)*3600) / 60)
	seconds = seconds - float64(hours)*3600 - float64(minutes)*60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, int(seconds))
}
func appendToFile(content string) {
	f, err := os.OpenFile("domains_ok.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	if _, err = f.WriteString(content + "\n"); err != nil {
		fmt.Println(err)
	}
}
