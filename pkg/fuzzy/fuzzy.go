package fuzzy

import "strings"

// Match reports whether text fuzzily matches query.
//
// Split query to words and check whether some words appear in in subsequence.
// gthb will match the github.com.
func Match(query, text string) bool {
	text = strings.ToLower(text)
	for word := range strings.FieldsSeq(strings.ToLower(query)) {
		if !subsequence(word, text) {
			return false
		}
	}
	return true
}

// subsequence reports whether the runes of needle appear in hay in order.
func subsequence(needle, hay string) bool {
	n := []rune(needle)
	i := 0
	for _, r := range hay {
		if i < len(n) && r == n[i] {
			i++
		}
	}
	return i == len(n)
}
