package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

var (
	file    string
	limit   int
	line    int
	count   int64
	total   int64
	wg      sync.WaitGroup
	help    bool
	verbose bool
	server  string
)

func init() {
	flag.StringVar(&file, "file", "domains.txt", "text file containing domain names")
	flag.IntVar(&limit, "limit", 10, "concurrency limit")
	flag.IntVar(&line, "line", 0, "start line")
	flag.BoolVar(&verbose, "v", false, "output nslookup results")
	flag.BoolVar(&help, "h", false, "show help information")
	flag.StringVar(&server, "server", "", "DNS server to use")
	flag.Parse()
}

func main() {
	if help {
		flag.Usage()
		os.Exit(0)
	}
	start := time.Now()
	var elapsed float64
	var average float64
	var estimate float64

	f, err := os.Open(file)
	if err != nil {
		flag.Usage()
		log.Fatal(err)
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
	var dnsserver string
	if server == "" {
		dnsserver = "system default"
	} else {
		dnsserver = server
	}
	fmt.Println("DNS Server:", dnsserver)
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
			fmt.Printf("\rProcessed %d/%d domains[%.4f%%]. Average time: %.2f seconds. Estimated total time: %s.", count, total, 100*float64(count)/float64(total), average, formatDuration(estimate))
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
		}
		close(ch)
	}()
	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range ch {
				now := time.Now()
				nslookup(domain)
				atomic.AddInt64(&count, 1)
				elapsed = now.Sub(start).Seconds()
				if count > 0 {
					average = elapsed / float64(count)
					estimate = float64(total-count) * average
				}
				fmt.Printf("\rProcessed %d/%d domains[%.4f%%]. Average time: %.2f seconds. Estimated total time: %s.", count, total, 100*float64(count)/float64(total), average, formatDuration(estimate))
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

func nslookup(domain string) {
	var cmd *exec.Cmd
	if server == "" {
		cmd = exec.Command("nslookup", domain)
	} else {
		cmd = exec.Command("nslookup", domain, server)
	}
	out, err := cmd.Output()
	if err != nil {
		//log.Println(err)
		return
	}
	if verbose {
		fmt.Println(string(out))
	}
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
