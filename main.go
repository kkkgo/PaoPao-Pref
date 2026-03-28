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
	grfile   string
	crfile   string
	cnfile   string
	comp     string
	inrule   string
	filter   string
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
	cndat    string
	cnmode   string
	skipfile string
	matcher  *CIDRMatcher
	skipRoot *trieNode
)
var domainRegex = regexp.MustCompile(`^[A-Za-z0-9.][a-zA-Z0-9.-]+\.[a-zA-Z]{2}[a-zA-Z]*$`)

var (
	outputFile *os.File
	outputMu   sync.Mutex
	statsMu    sync.Mutex
)

func init() {
	flag.StringVar(&file, "file", "domains.txt", "text file containing domain names")
	flag.StringVar(&gbfile, "gbfile", "", "comp")
	flag.StringVar(&grfile, "grfile", "", "comp")
	flag.StringVar(&crfile, "crfile", "", "comp")
	flag.StringVar(&cnfile, "cnfile", "", "comp")
	flag.StringVar(&comp, "comp", "", "comp gb cn result file.")
	flag.StringVar(&inrule, "inrule", "", "input proxy rule")
	flag.StringVar(&filter, "filter", "", "input filter rule")
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
	flag.StringVar(&cndat, "cndat", "", "path to CN-local.dat file")
	flag.StringVar(&cnmode, "cnmode", "", "CN verification mode: check|mark|cnmark")
	flag.StringVar(&skipfile, "skipfile", "", "path to skip list file (domains to skip in mark/cnmark modes)")
}

func main() {
	flag.Parse()
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
	if filter != "" {
		err := filterRules(inrule, filter, outrule)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
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
	if cndat != "" {
		var err error
		matcher, err = LoadDat(cndat)
		if err != nil {
			fmt.Println("Failed to load CN dat:", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded CN dat: %d CN, %d PRIVATE, %d CLOUDFLARE CIDRs\n", len(matcher.cn), len(matcher.private), len(matcher.cloudflare))
	}
	if cnmode != "" && matcher == nil {
		fmt.Println("cnmode requires -cndat")
		os.Exit(1)
	}
	if skipfile != "" {
		skipKeywords, err := readKeywords(skipfile)
		if err != nil {
			fmt.Println("Failed to read skipfile:", err)
			os.Exit(1)
		}
		skipRoot = newTrieNode()
		for _, kw := range skipKeywords {
			skipRoot.insert(splitDomainParts(kw))
		}
		fmt.Printf("Loaded skip list: %d domains\n", len(skipKeywords))
	}
	if delay {
		if check_delay("www.taobao.com") {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
	if output {
		var err error
		outputFile, err = os.OpenFile("domains_ok.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to open output file:", err)
			os.Exit(1)
		}
		defer outputFile.Close()
	}
	start := time.Now()
	var elapsed float64
	var average float64
	var estimate float64

	f, err := os.Open(file)
	if err != nil {
		flag.Usage()
		fmt.Println(err)
		os.Exit(1)
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
			if atomic.LoadInt64(&count) >= total {
				break
			}
			domain := scanner.Text()
			ch <- domain
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
				var ok bool
				if cnmode != "" {
					ok = cnQuery(domain)
				} else {
					ok = nslookup(domain)
				}
				if ok {
					atomic.AddInt64(&succ, 1)
				}
				currentCount := atomic.AddInt64(&count, 1)
				currentSucc := atomic.LoadInt64(&succ)

				statsMu.Lock()
				elapsed = time.Since(start).Seconds()
				if currentCount > 0 {
					average = elapsed / float64(currentCount)
					estimate = float64(total-currentCount) * average
				}
				avg := average
				est := estimate
				statsMu.Unlock()

				fmt.Printf("\rNslookup %d/%d domains[%.4f%%]. Succ rate:%.2f%%. Avg time: %.6f seconds. Est time: %s.", currentCount, total, 100*float64(currentCount)/float64(total), 100*float64(currentSucc)/float64(currentCount), avg, formatDuration(est))
				time.Sleep(sleep)
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
	var result bool
	if cnmode != "" {
		result = cnQuery(domain)
	} else {
		result = nslookup(domain)
	}
	elapsed := time.Since(start)
	if result {
		fmt.Printf("%d", elapsed.Milliseconds())
		return true
	} else {
		return false
	}
}

// cnQuery performs a DNS lookup and checks the result IPs against CN/PRIVATE/CLOUDFLARE CIDRs.
// Returns true/false based on cnmode:
//   - "check": true if any response IPv4 is CN
//   - "mark": true if resolved and NO response IP is CN/PRIVATE/CLOUDFLARE (global domain)
//   - "cnmark": true if any response IPv4 is CN
func cnQuery(domain string) bool {
	domain = strings.Replace(domain, "domain:.", "", 1)
	domain = strings.Replace(domain, "domain:", "", 1)

	if skipRoot != nil && skipRoot.hasAncestor(splitDomainParts(addDotIfMissing(domain))) {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	addrs, err := resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		if verbose {
			fmt.Printf("\n\033[33m%s\033[0m\n", err)
		}
		return false
	}

	// Filter to IPv4 only
	var ipv4s []net.IP
	for _, addr := range addrs {
		if ip4 := addr.IP.To4(); ip4 != nil {
			ipv4s = append(ipv4s, ip4)
		}
	}
	if len(ipv4s) == 0 {
		return false
	}

	if verbose {
		fmt.Printf("\n\033[31m%s\033[0m\033[32m%v\033[0m\n", domain, ipv4s)
	}

	switch cnmode {
	case "check":
		for _, ip := range ipv4s {
			if matcher.MatchCN(ip) {
				if output {
					appendToFile(domain)
				}
				return true
			}
		}
		return false
	case "mark":
		for _, ip := range ipv4s {
			if matcher.MatchCN(ip) || matcher.MatchPrivate(ip) || matcher.MatchCloudflare(ip) {
				return false
			}
		}
		if output {
			appendToFile(domain)
		}
		return true
	case "cnmark":
		for _, ip := range ipv4s {
			if matcher.MatchCN(ip) {
				if output {
					appendToFile(domain)
				}
				return true
			}
		}
		return false
	default:
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
	outputMu.Lock()
	defer outputMu.Unlock()
	if outputFile == nil {
		return
	}
	if _, err := outputFile.WriteString(content + "\n"); err != nil {
		fmt.Println(err)
	}
}

func convertRule(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" || strings.HasPrefix(rule, "//") || strings.HasPrefix(rule, "!") || strings.HasPrefix(rule, "@") || strings.HasPrefix(rule, "[") || strings.HasPrefix(rule, "full:") {
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
func filterRules(inputFile, filterFile, outputFile string) error {

	iRules, err := readKeywords(inputFile)
	if err != nil {
		return err
	}
	fRules, err := readKeywords(filterFile)
	if err != nil {
		return err
	}
	output, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer output.Close()
	fmt.Fprintln(output)

	fRoot := newTrieNode()
	for _, fKeyword := range fRules {
		fRoot.insert(splitDomainParts(fKeyword))
	}
	for _, iKeyword := range iRules {
		if !fRoot.hasAncestor(splitDomainParts(iKeyword)) {
			fmt.Fprintln(output, "domain:"+iKeyword)
		}
	}
	fmt.Println("inrule filter finish.")
	return nil
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

	for scanner.Scan() {
		line := scanner.Text()
		convertedRule := addDotIfMissing(convertRule(line))
		if convertedRule != "" {
			convertedRules[convertedRule] = true
		}
	}
	fmt.Println("read domains.")

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
type trieNode struct {
	children map[string]*trieNode
	isEnd    bool
}

func newTrieNode() *trieNode {
	return &trieNode{children: make(map[string]*trieNode)}
}

// hasAncestor checks if any ancestor (shorter suffix) is already marked as end.
func (t *trieNode) hasAncestor(parts []string) bool {
	node := t
	for _, part := range parts {
		child, ok := node.children[part]
		if !ok {
			return false
		}
		if child.isEnd {
			return true
		}
		node = child
	}
	return false
}

func (t *trieNode) insert(parts []string) {
	node := t
	for _, part := range parts {
		child, ok := node.children[part]
		if !ok {
			child = newTrieNode()
			node.children[part] = child
		}
		node = child
	}
	node.isEnd = true
}

// splitDomainParts splits a domain and returns reversed non-empty parts for trie operations.
func splitDomainParts(domain string) []string {
	raw := strings.Split(domain, ".")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		if p != "" {
			parts = append(parts, p)
		}
	}
	// Reverse so TLD comes first (suffix matching becomes prefix matching).
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}

func mergeDomains(domains []string) []string {
	seen := make(map[string]struct{}, len(domains))
	uniqueDomains := make([]string, 0, len(domains))
	for _, domain := range domains {
		if _, ok := seen[domain]; !ok {
			seen[domain] = struct{}{}
			uniqueDomains = append(uniqueDomains, domain)
		}
	}
	// Sort by number of non-empty parts (shorter domains first).
	sort.Slice(uniqueDomains, func(i, j int) bool {
		ci := strings.Count(strings.Trim(uniqueDomains[i], "."), ".")
		cj := strings.Count(strings.Trim(uniqueDomains[j], "."), ".")
		return ci < cj
	})

	root := newTrieNode()
	result := make([]string, 0, len(uniqueDomains))

	for _, domain := range uniqueDomains {
		parts := splitDomainParts(domain)
		if len(parts) == 0 {
			continue
		}
		if root.hasAncestor(parts) {
			continue
		}
		root.insert(parts)
		result = append(result, domain)
	}

	return result
}

func reverseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	var b strings.Builder
	b.Grow(len(domain))
	for i := len(parts) - 1; i >= 0; i-- {
		b.WriteString(parts[i])
		if i != 0 {
			b.WriteByte('.')
		}
	}
	return b.String()
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
	globalRules, err := readKeywords(grfile)
	if err != nil {
		fmt.Printf("read globalRules failed：%v\n", err)
		return 1
	}

	if err := processCNFile(cnfile, globalKeywords, globalRules, comp, gbfile, crfile); err != nil {
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
func processCNFile(cnFile string, globalKeywords []string, globalRules []string, resultFile string, globalFile string, cnruleFile string) error {
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
	cnr, err := os.Open(cnruleFile)
	if err != nil {
		return err
	}
	defer cnr.Close()

	_, err = io.Copy(result, global)
	if err != nil {
		return err
	}
	fmt.Fprintln(result)
	fmt.Fprintln(result)

	cnrScanner := bufio.NewScanner(cnr)
	for cnrScanner.Scan() {
		line := cnrScanner.Text()
		fmt.Fprintln(result, strings.Replace(line, "domain:", "#@domain:", 1))
	}
	fmt.Fprintln(result)
	// Build tries for fast suffix matching
	gkRoot := newTrieNode()
	for _, gk := range globalKeywords {
		gkRoot.insert(splitDomainParts(gk))
	}
	grRoot := newTrieNode()
	for _, gr := range globalRules {
		grRoot.insert(splitDomainParts(gr))
	}

	cnScanner := bufio.NewScanner(cn)
	for cnScanner.Scan() {
		line := cnScanner.Text()
		keyword := strings.TrimPrefix(line, "domain:")
		if gkRoot.hasAncestor(splitDomainParts(keyword)) && !grRoot.hasAncestor(splitDomainParts(keyword)) {
			fmt.Fprintln(result, strings.Replace(line, "domain:", "##@@domain:", 1))
		}
	}
	fmt.Fprintln(result)

	if err := cnScanner.Err(); err != nil {
		return err
	}

	return nil
}
