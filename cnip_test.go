package main

import (
	"encoding/binary"
	"net"
	"os"
	"testing"
)

// buildTestDat creates a minimal protobuf-encoded GeoIPList for testing.
// Schema: GeoIPList { repeated GeoIP entry = 1 }
//         GeoIP { string country_code = 1; repeated CIDR cidr = 2 }
//         CIDR { bytes ip = 1; uint32 prefix = 2 }
func buildTestDat() []byte {
	var buf []byte

	// Helper: encode varint
	encVarint := func(v uint64) []byte {
		b := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(b, v)
		return b[:n]
	}
	// Helper: encode a length-delimited field
	encLD := func(fieldNum uint64, data []byte) []byte {
		tag := encVarint((fieldNum << 3) | 2)
		length := encVarint(uint64(len(data)))
		out := make([]byte, 0, len(tag)+len(length)+len(data))
		out = append(out, tag...)
		out = append(out, length...)
		out = append(out, data...)
		return out
	}
	// Helper: encode a varint field
	encVF := func(fieldNum uint64, val uint64) []byte {
		tag := encVarint((fieldNum << 3) | 0)
		v := encVarint(val)
		out := make([]byte, 0, len(tag)+len(v))
		out = append(out, tag...)
		out = append(out, v...)
		return out
	}
	// Helper: encode CIDR message
	encCIDR := func(ip net.IP, prefix int) []byte {
		var msg []byte
		msg = append(msg, encLD(1, []byte(ip))...)
		msg = append(msg, encVF(2, uint64(prefix))...)
		return msg
	}
	// Helper: encode GeoIP message
	encGeoIP := func(code string, cidrs [][]byte) []byte {
		var msg []byte
		msg = append(msg, encLD(1, []byte(code))...)
		for _, c := range cidrs {
			msg = append(msg, encLD(2, c)...)
		}
		return msg
	}

	// CN entries: 114.114.114.0/24, 223.5.5.0/24, 119.29.29.0/24
	cnCIDRs := [][]byte{
		encCIDR(net.IP{114, 114, 114, 0}, 24),
		encCIDR(net.IP{223, 5, 5, 0}, 24),
		encCIDR(net.IP{119, 29, 29, 0}, 24),
	}
	cnEntry := encGeoIP("cn", cnCIDRs)

	// PRIVATE entries: 10.0.0.0/8, 192.168.0.0/16, 127.0.0.0/8
	privateCIDRs := [][]byte{
		encCIDR(net.IP{10, 0, 0, 0}, 8),
		encCIDR(net.IP{192, 168, 0, 0}, 16),
		encCIDR(net.IP{127, 0, 0, 0}, 8),
	}
	privateEntry := encGeoIP("private", privateCIDRs)

	// GeoIPList: field 1 repeated
	buf = append(buf, encLD(1, cnEntry)...)
	buf = append(buf, encLD(1, privateEntry)...)

	return buf
}

func loadTestMatcher(t *testing.T) *CIDRMatcher {
	t.Helper()
	data := buildTestDat()
	tmpFile := t.TempDir() + "/test.dat"
	if err := writeTestFile(tmpFile, data); err != nil {
		t.Fatal(err)
	}
	m, err := LoadDat(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func writeTestFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func TestParseDatFile(t *testing.T) {
	m := loadTestMatcher(t)

	if len(m.cn) != 3 {
		t.Errorf("expected 3 CN entries, got %d", len(m.cn))
	}
	if len(m.private) != 3 {
		t.Errorf("expected 3 PRIVATE entries, got %d", len(m.private))
	}
}

func TestCIDRMatcherMatchCN(t *testing.T) {
	m := loadTestMatcher(t)

	tests := []struct {
		ip       string
		expected bool
	}{
		{"114.114.114.114", true},
		{"223.5.5.5", true},
		{"119.29.29.29", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := m.MatchCN(ip)
		if got != tt.expected {
			t.Errorf("MatchCN(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestCIDRMatcherMatchPrivate(t *testing.T) {
	m := loadTestMatcher(t)

	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
		{"172.17.0.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := m.MatchPrivate(ip)
		if got != tt.expected {
			t.Errorf("MatchPrivate(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestCIDRMatcherMatchCNOrPrivate(t *testing.T) {
	m := loadTestMatcher(t)

	tests := []struct {
		ip       string
		expected bool
	}{
		{"114.114.114.114", true}, // CN
		{"10.0.0.1", true},       // Private
		{"8.8.8.8", false},       // Neither
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		got := m.MatchCNOrPrivate(ip)
		if got != tt.expected {
			t.Errorf("MatchCNOrPrivate(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestLoadDatInvalidFile(t *testing.T) {
	_, err := LoadDat("/nonexistent/path.dat")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadDatEmptyData(t *testing.T) {
	tmpFile := t.TempDir() + "/empty.dat"
	if err := writeTestFile(tmpFile, []byte{}); err != nil {
		t.Fatal(err)
	}
	_, err := LoadDat(tmpFile)
	if err == nil {
		t.Error("expected error for empty dat file (no CN entries)")
	}
}

func BenchmarkMatchCN(b *testing.B) {
	data := buildTestDat()
	tmpFile := b.TempDir() + "/bench.dat"
	if err := writeTestFile(tmpFile, data); err != nil {
		b.Fatal(err)
	}
	m, err := LoadDat(tmpFile)
	if err != nil {
		b.Fatal(err)
	}
	ip := net.ParseIP("114.114.114.114")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.MatchCN(ip)
	}
}

func BenchmarkMatchCNMiss(b *testing.B) {
	data := buildTestDat()
	tmpFile := b.TempDir() + "/bench.dat"
	if err := writeTestFile(tmpFile, data); err != nil {
		b.Fatal(err)
	}
	m, err := LoadDat(tmpFile)
	if err != nil {
		b.Fatal(err)
	}
	ip := net.ParseIP("8.8.8.8")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.MatchCN(ip)
	}
}
