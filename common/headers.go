package common

import (
	"fmt"
	"strings"
)

const hexDigits = "0123456789ABCDEF"

func isHeaderNameChar(c byte) bool {
	if c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
		return true
	}
	return strings.IndexByte("!#$&'*+-.^_`|~", c) >= 0
}

func percentEncode(sb *strings.Builder, c byte) {
	sb.WriteByte('%')
	sb.WriteByte(hexDigits[c>>4])
	sb.WriteByte(hexDigits[c&0xF])
}

// EncodeHeaderName percent-encodes characters of an xRegistry map-key that
// are not allowed in an HTTP header name (e.g. ':' -> %3A).
func EncodeHeaderName(name string) string {
	sb := strings.Builder{}
	for i := 0; i < len(name); i++ {
		if isHeaderNameChar(name[i]) {
			sb.WriteByte(name[i])
		} else {
			percentEncode(&sb, name[i])
		}
	}
	return sb.String()
}

// EncodeHeaderValue encodes a value per the xRegistry HTTP header rules:
// non-printable/non-US-ASCII, '%' and '"' are percent encoded, as are
// leading/trailing spaces (so they survive trimming). Values with commas
// are wrapped in a double-quoted group.
func EncodeHeaderValue(val string) string {
	sb := strings.Builder{}
	start := len(val) - len(strings.TrimLeft(val, " \t"))
	end := len(strings.TrimRight(val, " \t"))
	for i := 0; i < len(val); i++ {
		c := val[i]
		edge := c == ' ' && (i < start || i >= end)
		if c < 0x20 || c > 0x7E || c == '%' || c == '"' || edge {
			percentEncode(&sb, c)
		} else {
			sb.WriteByte(c)
		}
	}
	res := sb.String()
	if strings.Contains(res, ",") {
		res = `"` + res + `"`
	}
	return res
}

// DecodeHeaderValue reverses EncodeHeaderValue: trims surrounding
// whitespace, removes double-quote group markers and percent-decodes.
func DecodeHeaderValue(val string) (string, error) {
	val = strings.Trim(val, " \t")
	sb := []byte{}
	inGroup := false
	for i := 0; i < len(val); i++ {
		c := val[i]
		switch {
		case c == '"':
			inGroup = !inGroup
		case c == '%':
			if i+2 >= len(val) {
				return "", fmt.Errorf("invalid percent-encoding at %q",
					val[i:])
			}
			h, l := unhex(val[i+1]), unhex(val[i+2])
			if h < 0 || l < 0 {
				return "", fmt.Errorf("invalid percent-encoding at %q",
					val[i:i+3])
			}
			sb = append(sb, byte(h<<4|l))
			i += 2
		default:
			sb = append(sb, c)
		}
	}
	if inGroup {
		return "", fmt.Errorf("unterminated double-quote")
	}
	return string(sb), nil
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
