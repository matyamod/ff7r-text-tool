package core

import (
	"unicode"
	"unicode/utf16"
)

// Convert UTF16 buffer to go string
func UTF16BytesToStr(buf []byte) string {
	utf16Units := make([]uint16, len(buf)/2)
	for i := 0; i < len(buf); i += 2 {
		utf16Units[i/2] = uint16(buf[i]) | uint16(buf[i+1])<<8
	}
	runes := utf16.Decode(utf16Units)
	return string(runes)
}

// Convert go string to UTF16 buffer
func StrToUTF16Bytes(str string) ([]byte, int) {
	utf16Units := utf16.Encode([]rune(str))
	utf16Bytes := make([]byte, len(utf16Units)*2) // each uint16 is 2 bytes

	for i, unit := range utf16Units {
		utf16Bytes[i*2] = byte(unit)        // lower byte
		utf16Bytes[i*2+1] = byte(unit >> 8) // upper byte
	}
	return utf16Bytes, len(utf16Units)
}

// Get length of UTF16 string from go string
func GetUTF16Len(str string) int {
	utf16Units := utf16.Encode([]rune(str))
	return len(utf16Units)
}

// Return false when a string contains non-ascii characters
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
