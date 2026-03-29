package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type CIDRMatcher struct {
	cn      []*net.IPNet
	private []*net.IPNet
}

// LoadDat reads a v2ray-format GeoIPList protobuf file and returns a CIDRMatcher.
func LoadDat(path string) (*CIDRMatcher, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dat file: %w", err)
	}
	entries, err := parseGeoIPList(data)
	if err != nil {
		return nil, fmt.Errorf("parse dat file: %w", err)
	}
	m := &CIDRMatcher{}
	for _, entry := range entries {
		code := strings.ToLower(entry.countryCode)
		nets := make([]*net.IPNet, 0, len(entry.cidrs))
		for _, c := range entry.cidrs {
			ipNet := c.toIPNet()
			if ipNet != nil {
				nets = append(nets, ipNet)
			}
		}
		switch code {
		case "cn":
			m.cn = nets
		case "private":
			m.private = nets
		}
	}
	if len(m.cn) == 0 {
		return nil, fmt.Errorf("no CN entries found in dat file")
	}
	if len(m.private) == 0 {
		return nil, fmt.Errorf("no PRIVATE entries found in dat file")
	}
	return m, nil
}

func (m *CIDRMatcher) MatchCN(ip net.IP) bool {
	return matchNets(ip, m.cn)
}

func (m *CIDRMatcher) MatchPrivate(ip net.IP) bool {
	return matchNets(ip, m.private)
}

func (m *CIDRMatcher) MatchCNOrPrivate(ip net.IP) bool {
	return m.MatchCN(ip) || m.MatchPrivate(ip)
}

func matchNets(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// --- Protobuf wire format parser for GeoIPList ---
// Schema:
//   message CIDR { bytes ip = 1; uint32 prefix = 2; }
//   message GeoIP { string country_code = 1; repeated CIDR cidr = 2; }
//   message GeoIPList { repeated GeoIP entry = 1; }

type geoIPEntry struct {
	countryCode string
	cidrs       []cidrEntry
}

type cidrEntry struct {
	ip     []byte
	prefix uint32
}

func (c *cidrEntry) toIPNet() *net.IPNet {
	if len(c.ip) != 4 && len(c.ip) != 16 {
		return nil
	}
	bits := 32
	if len(c.ip) == 16 {
		bits = 128
	}
	return &net.IPNet{
		IP:   net.IP(c.ip),
		Mask: net.CIDRMask(int(c.prefix), bits),
	}
}

// readVarint reads a protobuf varint from data[pos:] and returns (value, newPos).
func readVarint(data []byte, pos int) (uint64, int, error) {
	var val uint64
	var shift uint
	for pos < len(data) {
		b := data[pos]
		pos++
		val |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return val, pos, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, pos, fmt.Errorf("varint overflow")
		}
	}
	return 0, pos, fmt.Errorf("unexpected end of data reading varint")
}

// parseGeoIPList parses the top-level GeoIPList message.
func parseGeoIPList(data []byte) ([]geoIPEntry, error) {
	var entries []geoIPEntry
	pos := 0
	for pos < len(data) {
		tag, newPos, err := readVarint(data, pos)
		if err != nil {
			return nil, err
		}
		pos = newPos
		fieldNum := tag >> 3
		wireType := tag & 0x07

		if wireType != 2 {
			// Skip non-length-delimited fields
			pos, err = skipField(data, pos, wireType)
			if err != nil {
				return nil, err
			}
			continue
		}

		length, newPos, err := readVarint(data, pos)
		if err != nil {
			return nil, err
		}
		pos = newPos
		end := pos + int(length)
		if end > len(data) {
			return nil, fmt.Errorf("length exceeds data")
		}

		if fieldNum == 1 { // repeated GeoIP entry
			entry, err := parseGeoIP(data[pos:end])
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
		pos = end
	}
	return entries, nil
}

// parseGeoIP parses a GeoIP message.
func parseGeoIP(data []byte) (geoIPEntry, error) {
	var entry geoIPEntry
	pos := 0
	for pos < len(data) {
		tag, newPos, err := readVarint(data, pos)
		if err != nil {
			return entry, err
		}
		pos = newPos
		fieldNum := tag >> 3
		wireType := tag & 0x07

		if wireType == 2 {
			length, newPos, err := readVarint(data, pos)
			if err != nil {
				return entry, err
			}
			pos = newPos
			end := pos + int(length)
			if end > len(data) {
				return entry, fmt.Errorf("length exceeds data")
			}
			switch fieldNum {
			case 1: // country_code (string)
				entry.countryCode = string(data[pos:end])
			case 2: // repeated CIDR
				c, err := parseCIDR(data[pos:end])
				if err != nil {
					return entry, err
				}
				entry.cidrs = append(entry.cidrs, c)
			}
			pos = end
		} else if wireType == 0 {
			_, newPos, err := readVarint(data, pos)
			if err != nil {
				return entry, err
			}
			pos = newPos
		} else {
			pos, err = skipField(data, pos, wireType)
			if err != nil {
				return entry, err
			}
		}
	}
	return entry, nil
}

// parseCIDR parses a CIDR message.
func parseCIDR(data []byte) (cidrEntry, error) {
	var c cidrEntry
	pos := 0
	for pos < len(data) {
		tag, newPos, err := readVarint(data, pos)
		if err != nil {
			return c, err
		}
		pos = newPos
		fieldNum := tag >> 3
		wireType := tag & 0x07

		if wireType == 2 {
			length, newPos, err := readVarint(data, pos)
			if err != nil {
				return c, err
			}
			pos = newPos
			end := pos + int(length)
			if end > len(data) {
				return c, fmt.Errorf("length exceeds data")
			}
			if fieldNum == 1 { // ip (bytes)
				c.ip = make([]byte, length)
				copy(c.ip, data[pos:end])
			}
			pos = end
		} else if wireType == 0 {
			val, newPos, err := readVarint(data, pos)
			if err != nil {
				return c, err
			}
			pos = newPos
			if fieldNum == 2 { // prefix (uint32)
				c.prefix = uint32(val)
			}
		} else {
			pos, err = skipField(data, pos, wireType)
			if err != nil {
				return c, err
			}
		}
	}
	return c, nil
}

// skipField skips a protobuf field value based on wire type.
func skipField(data []byte, pos int, wireType uint64) (int, error) {
	switch wireType {
	case 0: // varint
		_, newPos, err := readVarint(data, pos)
		return newPos, err
	case 1: // 64-bit
		return pos + 8, nil
	case 2: // length-delimited
		length, newPos, err := readVarint(data, pos)
		if err != nil {
			return pos, err
		}
		return newPos + int(length), nil
	case 5: // 32-bit
		return pos + 4, nil
	default:
		return pos, fmt.Errorf("unknown wire type %d", wireType)
	}
}
