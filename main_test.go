package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
)

// ============================================================
// Unit Tests
// ============================================================

func TestConvertRule(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Skip rules
		{"", ""},
		{"// comment", ""},
		{"! comment", ""},
		{"@rule", ""},
		{"[section]", ""},
		{"full:example.com", ""},

		// domain: prefix
		{"domain:example.com", "example.com"},
		{"domain:.example.com", ".example.com"},

		// Plain domain
		{"example.com", "example.com"},
		{"sub.example.com", "sub.example.com"},

		// || prefix
		{"||example.com", "example.com"},
		{"||*.example.com", ".example.com"},

		// | prefix with URL
		{"|https://example.com/path", "example.com"},

		// Whitespace trimming
		{"  example.com  ", "example.com"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
			result := convertRule(tt.input)
			if result != tt.expected {
				t.Errorf("convertRule(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddDotIfMissing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{".example.com", ".example.com"},
		{"example.com", ".example.com"},
	}

	for _, tt := range tests {
		result := addDotIfMissing(tt.input)
		if result != tt.expected {
			t.Errorf("addDotIfMissing(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractMainDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"www.example.com", "example.com"},
		{"example.com", "example.com"},
		{"a.b.c.example.com", "example.com"},
		{"localhost", "localhost"},
	}

	for _, tt := range tests {
		result := extractMainDomain(tt.input)
		if result != tt.expected {
			t.Errorf("extractMainDomain(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestReverseDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"www.example.com", "com.example.www"},
		{"example.com", "com.example"},
		{"localhost", "localhost"},
	}

	for _, tt := range tests {
		result := reverseDomain(tt.input)
		if result != tt.expected {
			t.Errorf("reverseDomain(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  float64
		expected string
	}{
		{0, "00:00:00"},
		{61, "00:01:01"},
		{3661, "01:01:01"},
		{7200, "02:00:00"},
		{90.5, "00:01:30"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDuration(%f) = %q, want %q", tt.seconds, result, tt.expected)
		}
	}
}

func TestMergeDomains(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no merge needed",
			input:    []string{".example.com", ".google.com"},
			expected: []string{".example.com", ".google.com"},
		},
		{
			name:     "subdomain merged into parent",
			input:    []string{".example.com", ".sub.example.com", ".deep.sub.example.com"},
			expected: []string{".example.com"},
		},
		{
			name:     "duplicates removed",
			input:    []string{".example.com", ".example.com"},
			expected: []string{".example.com"},
		},
		{
			name:     "mixed merge",
			input:    []string{".a.example.com", ".example.com", ".google.com", ".mail.google.com"},
			expected: []string{".example.com", ".google.com"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeDomains(tt.input)
			sort.Strings(result)
			expected := make([]string, len(tt.expected))
			copy(expected, tt.expected)
			sort.Strings(expected)

			if len(result) == 0 && len(expected) == 0 {
				return
			}
			if len(result) != len(expected) {
				t.Errorf("mergeDomains() returned %d items, want %d\ngot:  %v\nwant: %v", len(result), len(expected), result, expected)
				return
			}
			for i := range result {
				if result[i] != expected[i] {
					t.Errorf("mergeDomains()[%d] = %q, want %q", i, result[i], expected[i])
				}
			}
		})
	}
}

func TestTrieNode(t *testing.T) {
	root := newTrieNode()

	// Insert ["com", "example"] (reversed parts of "example.com")
	root.insert([]string{"com", "example"})

	// "www.example.com" → reversed ["com", "example", "www"] should have ancestor
	if !root.hasAncestor([]string{"com", "example", "www"}) {
		t.Error("expected hasAncestor=true for com.example.www")
	}

	// "google.com" → reversed ["com", "google"] should not have ancestor
	if root.hasAncestor([]string{"com", "google"}) {
		t.Error("expected hasAncestor=false for com.google")
	}

	// "com" alone should not match
	if root.hasAncestor([]string{"com"}) {
		t.Error("expected hasAncestor=false for com")
	}

	// hasAncestor checks child.isEnd before descending to next level.
	// For exact match ["com", "example"], "example" child's isEnd=true is found.
	if !root.hasAncestor([]string{"com", "example"}) {
		t.Error("expected hasAncestor=true for exact match com.example")
	}
}

func TestConvertRules(t *testing.T) {
	inputContent := `domain:example.com
domain:sub.example.com
||google.com
! comment
// another comment
test.org
`
	inputFile := t.TempDir() + "/input.txt"
	outputFile := t.TempDir() + "/output.txt"

	if err := os.WriteFile(inputFile, []byte(inputContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := convertRules(inputFile, outputFile)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Should contain merged domains
	if !strings.Contains(content, "domain:") {
		t.Error("output should contain domain: prefixed rules")
	}
	// Parent domain should subsume subdomain
	if strings.Contains(content, "sub.example.com") {
		t.Error("sub.example.com should be merged into example.com")
	}
}

func TestFilterRules(t *testing.T) {
	inputContent := "domain:.example.com\ndomain:.google.com\ndomain:.test.org\n"
	filterContent := "domain:.example.com\n"
	inputFile := t.TempDir() + "/input.txt"
	filterFile := t.TempDir() + "/filter.txt"
	outputFile := t.TempDir() + "/output.txt"

	os.WriteFile(inputFile, []byte(inputContent), 0644)
	os.WriteFile(filterFile, []byte(filterContent), 0644)

	err := filterRules(inputFile, filterFile, outputFile)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if strings.Contains(content, "example.com") {
		t.Error("example.com should be filtered out")
	}
	if !strings.Contains(content, "google.com") {
		t.Error("google.com should remain")
	}
	if !strings.Contains(content, "test.org") {
		t.Error("test.org should remain")
	}
}

func TestReadKeywords(t *testing.T) {
	content := "domain:.example.com\ndomain:.google.com\n\n  \n"
	file := t.TempDir() + "/keywords.txt"
	os.WriteFile(file, []byte(content), 0644)

	keywords, err := readKeywords(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(keywords))
	}
	if keywords[0] != ".example.com" {
		t.Errorf("expected .example.com, got %s", keywords[0])
	}
}

func TestAppendToFile(t *testing.T) {
	file := t.TempDir() + "/test_output.txt"
	var err error
	outputFile, err = os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}

	appendToFile("example.com")
	appendToFile("google.com")
	outputFile.Close()

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "example.com\n") {
		t.Error("should contain example.com")
	}
	if !strings.Contains(content, "google.com\n") {
		t.Error("should contain google.com")
	}

	// Reset global
	outputFile = nil
}

func TestAppendToFileNilSafe(t *testing.T) {
	outputFile = nil
	// Should not panic
	appendToFile("test.com")
}

// ============================================================
// Benchmarks — old vs new implementations
// ============================================================

// --- reverseDomain: old (string concat) vs new (strings.Builder) ---

func reverseDomainOld(domain string) string {
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

func BenchmarkReverseDomainOld(b *testing.B) {
	domains := []string{"www.example.com", "a.b.c.d.e.example.com", "very.deep.subdomain.test.example.co.uk"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range domains {
			reverseDomainOld(d)
		}
	}
}

func BenchmarkReverseDomainNew(b *testing.B) {
	domains := []string{"www.example.com", "a.b.c.d.e.example.com", "very.deep.subdomain.test.example.co.uk"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range domains {
			reverseDomain(d)
		}
	}
}

// --- mergeDomains: old (O(n^2)) vs new (Trie) ---

func mergeDomainsOld(domains []string) []string {
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

func generateTestDomains(n int) []string {
	tlds := []string{".com", ".org", ".net", ".io", ".dev"}
	parents := make([]string, 0, n/5)
	domains := make([]string, 0, n)

	for i := 0; i < n/5; i++ {
		parent := fmt.Sprintf(".domain%d%s", i, tlds[rand.Intn(len(tlds))])
		parents = append(parents, parent)
		domains = append(domains, parent)
	}

	for len(domains) < n {
		parent := parents[rand.Intn(len(parents))]
		sub := fmt.Sprintf(".sub%d%s", rand.Intn(1000), parent)
		domains = append(domains, sub)
	}

	return domains
}

func BenchmarkMergeDomainsOld_100(b *testing.B) {
	domains := generateTestDomains(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDomainsOld(domains)
	}
}

func BenchmarkMergeDomainsNew_100(b *testing.B) {
	domains := generateTestDomains(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDomains(domains)
	}
}

func BenchmarkMergeDomainsOld_1000(b *testing.B) {
	domains := generateTestDomains(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDomainsOld(domains)
	}
}

func BenchmarkMergeDomainsNew_1000(b *testing.B) {
	domains := generateTestDomains(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDomains(domains)
	}
}

func BenchmarkMergeDomainsOld_10000(b *testing.B) {
	domains := generateTestDomains(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDomainsOld(domains)
	}
}

func BenchmarkMergeDomainsNew_10000(b *testing.B) {
	domains := generateTestDomains(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeDomains(domains)
	}
}

// --- appendToFile: old (open/close each time) vs new (single handle) ---

func appendToFileOld(content string) {
	f, err := os.OpenFile("domains_ok_bench.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(content + "\n")
}

func BenchmarkAppendToFileOld(b *testing.B) {
	defer os.Remove("domains_ok_bench.txt")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appendToFileOld("example.com")
	}
}

func BenchmarkAppendToFileNew(b *testing.B) {
	file := b.TempDir() + "/bench_output.txt"
	var err error
	outputFile, err = os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		outputFile.Close()
		outputFile = nil
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		appendToFile("example.com")
	}
}

// --- convertRule: benchmark to confirm goroutine overhead ---

func BenchmarkConvertRuleSerial(b *testing.B) {
	rules := []string{
		"domain:example.com",
		"||google.com",
		"|https://test.org/path",
		"plain.domain.com",
		"! comment",
		"// comment",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, r := range rules {
			addDotIfMissing(convertRule(r))
		}
	}
}

// --- filterRules: old (O(n*m)) vs new (Trie) ---

func filterRulesOld(iRules, fRules []string) []string {
	var result []string
	for _, iKeyword := range iRules {
		global := true
		for _, fKeyword := range fRules {
			if strings.HasSuffix(iKeyword, fKeyword) {
				global = false
				break
			}
		}
		if global {
			result = append(result, iKeyword)
		}
	}
	return result
}

func filterRulesNew(iRules, fRules []string) []string {
	fRoot := newTrieNode()
	for _, fKeyword := range fRules {
		fRoot.insert(splitDomainParts(fKeyword))
	}
	var result []string
	for _, iKeyword := range iRules {
		if !fRoot.hasAncestor(splitDomainParts(iKeyword)) {
			result = append(result, iKeyword)
		}
	}
	return result
}

func generateFilterTestData(nInput, nFilter int) ([]string, []string) {
	tlds := []string{".com", ".org", ".net"}
	iRules := make([]string, 0, nInput)
	fRules := make([]string, 0, nFilter)

	for i := 0; i < nFilter; i++ {
		fRules = append(fRules, fmt.Sprintf(".filter%d%s", i, tlds[i%len(tlds)]))
	}
	for i := 0; i < nInput; i++ {
		if i%3 == 0 && len(fRules) > 0 {
			// Some inputs that match filter
			parent := fRules[i%len(fRules)]
			iRules = append(iRules, fmt.Sprintf(".sub%d%s", i, parent))
		} else {
			iRules = append(iRules, fmt.Sprintf(".domain%d%s", i, tlds[i%len(tlds)]))
		}
	}
	return iRules, fRules
}

func BenchmarkFilterRulesOld_1000x100(b *testing.B) {
	iRules, fRules := generateFilterTestData(1000, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterRulesOld(iRules, fRules)
	}
}

func BenchmarkFilterRulesNew_1000x100(b *testing.B) {
	iRules, fRules := generateFilterTestData(1000, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterRulesNew(iRules, fRules)
	}
}
