package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	file     string
	gbfile   string
	cnfile   string
	comp     string
	inrule   string
	outrule  string
	limit    int
	line     int
	pc       int
	count    int64
	succ     int64
	total    int64
	wg       sync.WaitGroup
	help     bool
	analyze  bool
	verbose  bool
	delay    bool
	server   string
	port     int
	output   bool
	timeout  time.Duration
	sleep    time.Duration
	resolver *net.Resolver
)
var domainRegex = regexp.MustCompile(`^[A-Za-z0-9.][a-zA-Z0-9.-]+\.[a-zA-Z]{2}[a-zA-Z]*$`)

func init() {
	flag.StringVar(&file, "file", "domains.txt", "text file containing domain names")
	flag.StringVar(&gbfile, "gbfile", "", "comp")
	flag.StringVar(&cnfile, "cnfile", "", "comp")
	flag.StringVar(&comp, "comp", "", "comp gb cn result file.")
	flag.StringVar(&inrule, "inrule", "", "input proxy rule")
	flag.StringVar(&outrule, "outrule", "", "output proxy rule.")
	flag.IntVar(&limit, "limit", 10, "concurrency limit")
	flag.IntVar(&line, "line", 0, "start line")
	flag.IntVar(&pc, "pc", 0, "test percentage")
	flag.BoolVar(&verbose, "v", false, "output nslookup results")
	flag.BoolVar(&delay, "delay", false, "check dns server delay")
	flag.BoolVar(&help, "h", false, "show help information")
	flag.BoolVar(&analyze, "an", false, "analyze")
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
	if comp != "" {
		os.Exit(comp_dat())
	}
	if analyze {
		inputFile := inrule
		anaOutputFile := outrule
		if inputFile == "" || anaOutputFile == "" {
			fmt.Println("-an -inrule domains.txt -outrule ana.txt")
			os.Exit(1)
		}
		input, err := os.Open(inputFile)
		if err != nil {
			fmt.Println("o err:", err)
			return
		}
		defer input.Close()

		anaOutput, err := os.Create(anaOutputFile)
		if err != nil {
			fmt.Println("w err:", err)
			return
		}
		defer anaOutput.Close()

		domainCount := make(map[string]int)

		scanner := bufio.NewScanner(input)
		for scanner.Scan() {
			domain := scanner.Text()
			mainDomain := extractMainDomain(domain)
			domainCount[mainDomain]++
		}
		if scanner.Err() != nil {
			fmt.Println("r err:", scanner.Err())
			return
		}
		type domainCountPair struct {
			Domain string
			Count  int
		}
		var domainCounts []domainCountPair
		for domain, count := range domainCount {
			domainCounts = append(domainCounts, domainCountPair{Domain: domain, Count: count})
		}
		sort.Slice(domainCounts, func(i, j int) bool {
			return domainCounts[i].Count > domainCounts[j].Count
		})
		for _, pair := range domainCounts {
			_, err := fmt.Fprintf(anaOutput, "%s: %d\n", pair.Domain, pair.Count)
			if err != nil {
				fmt.Println("w err:", err)
				return
			}
		}
		fmt.Println("analyze save:", anaOutputFile)
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
	if envPC, ok := os.LookupEnv("DNS_PC"); ok {
		if percentage, err := strconv.Atoi(envPC); err == nil {
			pc = percentage
		}
	}
	if pc <= 0 || pc > 100 {
		pc = 100
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
	if inrule != "" {
		err := convertRules(inrule, outrule)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
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
	if delay {
		if check_delay("www.taobao.com") {
			os.Exit(0)
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
	total = int64(float64(total) * float64(pc) * 0.01)
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Percentage :", pc, "%")
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
			if count >= total {
				break
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
	domain = strings.Replace(domain, "domain:.", "", 1)
	domain = strings.Replace(domain, "domain:", "", 1)
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

func check_delay(domain string) bool {
	start := time.Now()
	result := nslookup(domain)
	elapsed := time.Since(start)
	if result {
		fmt.Printf("%d", elapsed.Milliseconds())
		return true
	} else {
		return false
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

func convertRule(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" || strings.HasPrefix(rule, "//") || strings.HasPrefix(rule, "!") || strings.HasPrefix(rule, "@") || strings.HasPrefix(rule, "[") {
		return ""
	}

	if strings.HasPrefix(rule, "domain:") {
		return strings.Replace(rule, "domain:", "", 1)
	}

	match := domainRegex.MatchString(rule)
	if match {
		return rule
	}

	if len(rule) >= 2 && rule[:2] == "||" {
		domain := rule[2:]
		if strings.Contains(domain, "*") {
			return strings.TrimPrefix(domain, "*")
		}
		return domain
	} else if len(rule) >= 1 && rule[0] == '|' {
		u, err := url.Parse(rule[1:])
		if err != nil {
			fmt.Printf("Error parsing URL: %s\n", rule[1:])
			return ""
		}
		domain := u.Hostname()
		if strings.Contains(domain, "*") {
			return strings.TrimPrefix(domain, "*")
		}
		return domain
	}
	return ""
}
func addDotIfMissing(str string) string {
	if str == "" {
		return ""
	}

	if str[0] != '.' {
		str = "." + str
	}
	return str
}
func convertRules(inputFile, outputFile string) error {
	input, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()

	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	convertedRules := make(map[string]bool)
	batchSize := 1000000

	for {
		lines := make([]string, 0, batchSize)
		for i := 0; i < batchSize && scanner.Scan(); i++ {
			line := scanner.Text()
			lines = append(lines, line)
		}
		fmt.Println("read domains.")
		var wg sync.WaitGroup
		mu := sync.Mutex{}

		for _, line := range lines {
			wg.Add(1)
			go func(line string) {
				defer wg.Done()
				convertedRule := addDotIfMissing(convertRule(line))
				if convertedRule != "" {
					mu.Lock()
					convertedRules[convertedRule] = true
					mu.Unlock()
				}
			}(line)
		}

		wg.Wait()

		if len(lines) < batchSize {
			break
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}
	domainList := make([]string, 0, len(convertedRules))
	for do := range convertedRules {
		domainList = append(domainList, do)
	}
	fmt.Println("start mergedDomains.")
	mergedDomains := mergeDomains(domainList)
	for _, domain := range mergedDomains {
		_, err := fmt.Fprintln(writer, "domain:"+domain)
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(writer, "")
	err = writer.Flush()
	if err != nil {
		return err
	}
	fmt.Println("Conversion completed successfully.")
	return nil
}
func mergeDomains(domains []string) []string {
	result := make([]string, 0, len(domains))
	uniqueDomains := make([]string, 0, len(domains))
	seen := make(map[string]struct{})
	for _, domain := range domains {
		if _, ok := seen[domain]; !ok {
			seen[domain] = struct{}{}
			uniqueDomains = append(uniqueDomains, domain)
		}
	}
	sort.Slice(uniqueDomains, func(i, j int) bool {
		return len(uniqueDomains[i]) < len(uniqueDomains[j])
	})

	for _, domain := range uniqueDomains {
		found := false
		for _, uniqueDomain := range result {
			if strings.HasSuffix(domain, uniqueDomain) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, domain)
		}
	}

	return result
}

func reverseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	reversed := ""
	for i := len(parts) - 1; i >= 0; i-- {
		reversed += parts[i]
		if i != 0 {
			reversed += "."
		}
	}
	return reversed
}

func extractMainDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

func comp_dat() int {
	globalKeywords, err := readKeywords(gbfile)
	if err != nil {
		fmt.Printf("read global failed：%v\n", err)
		return 1
	}

	if err := processCNFile(cnfile, globalKeywords, comp, gbfile); err != nil {
		fmt.Printf("comp cn failed：%v\n", err)
		return 1
	}

	fmt.Println("comp finish!")
	return 0
}

func readKeywords(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keywords []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		keyword := strings.TrimPrefix(line, "domain:")
		keywords = append(keywords, keyword)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keywords, nil
}
func processCNFile(cnFile string, globalKeywords []string, resultFile string, globalFile string) error {
	cn, err := os.Open(cnFile)
	if err != nil {
		return err
	}
	defer cn.Close()

	result, err := os.Create(resultFile)
	if err != nil {
		return err
	}
	fmt.Fprintln(result)

	defer result.Close()

	global, err := os.Open(globalFile)
	if err != nil {
		return err
	}
	defer global.Close()

	_, err = io.Copy(result, global)
	if err != nil {
		return err
	}
	fmt.Fprintln(result)
	fmt.Fprintln(result)

	cnScanner := bufio.NewScanner(cn)
	for cnScanner.Scan() {
		line := cnScanner.Text()
		keyword := strings.TrimPrefix(line, "domain:")

		found := false
		for _, globalKeyword := range globalKeywords {
			if strings.Contains(keyword, globalKeyword) {
				found = true
				break
			}
		}

		if found {
			line = strings.Replace(line, "domain:", "##@@domain:", 1)
			fmt.Fprintln(result, line)
		}
	}
	fmt.Fprintln(result)

	if err := cnScanner.Err(); err != nil {
		return err
	}

	return nil
}
