package util

import (
	"regexp"
	"strconv"
	"strings"
)

var tokenizer = regexp.MustCompile(`(\d+|\D+)`)

type naturalSortToken struct {
	str   string
	num   int
	isNum bool
}

func tokenize(s string) []naturalSortToken {
	parts := tokenizer.FindAllString(s, -1)
	tokens := make([]naturalSortToken, len(parts))
	for i, p := range parts {
		num, err := strconv.Atoi(p)
		if err == nil {
			tokens[i] = naturalSortToken{num: num, isNum: true}
		} else {
			tokens[i] = naturalSortToken{str: strings.ToLower(p), isNum: false}
		}
	}
	return tokens
}

// Less compares two strings for natural sorting order.
func NaturalSortLess(s1, s2 string) bool {
	t1 := tokenize(s1)
	t2 := tokenize(s2)
	minLen := min(len(t1), len(t2))

	for i := 0; i < minLen; i++ {
		// If one is a number and the other isn't, the number comes first.
		if t1[i].isNum && !t2[i].isNum {
			return true
		}
		if !t1[i].isNum && t2[i].isNum {
			return false
		}

		if t1[i].isNum { // Both are numbers
			if t1[i].num != t2[i].num {
				return t1[i].num < t2[i].num
			}
		} else { // Both are strings
			if t1[i].str != t2[i].str {
				return t1[i].str < t2[i].str
			}
		}
	}

	// If all tokens so far are equal, the shorter string comes first.
	return len(t1) < len(t2)
}
