package main

import (
	"strings"
	"unicode"
)

func distortText(text string) string {
	count := 0
	return strings.Map(func(r rune) rune {
		count++ // index in `i, r := range text` counts +2 for 2-byte symbols, so count separate count is needed anyway
		if count%2 == 0 {
			return unicode.ToUpper(r)
		}
		return unicode.ToLower(r)
	}, text)
}
