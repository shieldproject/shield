package vault

import (
	"bytes"
	"strings"
)

func Encode(s string, n int) string {
	var buffer bytes.Buffer
	for i, rune := range s {
		buffer.WriteRune(rune)
		if i%n == (n-1) && i != (len(s)-1) {
			buffer.WriteRune('-')
		}
	}
	return buffer.String()
}

func Decode(s string) string {
	return strings.Replace(s, "-", "", -1)
}
