package util

import (
	"math/big"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// scanRe matches: key (non-digit chars not space), optional space, int or float.
var scanRe = regexp.MustCompile(`([^0-9\n\r ]*)[ ]*([0-9]*\.?[0-9]+)`)

// Item represents a parsed item with keys and their numeric values.
type Item struct {
	Numbers map[string]*big.Float
}

// NewItem creates a new Item.
func NewItem(numbers map[string]*big.Float) *Item {
	return &Item{Numbers: numbers}
}

// Compare compares this Item with another using the given keys.
// Returns -1 if this < other, 1 if this > other, 0 if equal.
func (i *Item) Compare(other *Item, keys []string) int {
	for _, key := range keys {
		aVal, aOk := i.Numbers[key]
		bVal, bOk := other.Numbers[key]

		if !aOk && !bOk {
			continue
		} else if !aOk {
			return 1
		} else if !bOk {
			return -1
		}

		cmp := aVal.Cmp(bVal)
		if cmp == 0 {
			continue
		}
		return cmp
	}
	return 0
}

// KeyRange tracks min, max and count of values for a key.
type KeyRange struct {
	Min   *big.Float
	Max   *big.Float
	Count int
}

// NewKeyRange initializes KeyRange with a value.
func NewKeyRange(value *big.Float) *KeyRange {
	return &KeyRange{
		Min:   new(big.Float).Set(value),
		Max:   new(big.Float).Set(value),
		Count: 1,
	}
}

// Update updates the KeyRange with a new value.
func (kr *KeyRange) Update(value *big.Float) {
	if value.Cmp(kr.Min) < 0 {
		kr.Min.Set(value)
	}
	if value.Cmp(kr.Max) > 0 {
		kr.Max.Set(value)
	}
	kr.Count++
}

// Range returns the difference between max and min.
func (kr *KeyRange) Range() *big.Float {
	r := new(big.Float).Sub(kr.Max, kr.Min)
	return r
}

// ChapterSorter sorts chapter strings respecting keys like "Vol." and "Ch."
type ChapterSorter struct {
	sortedKeys []string
}

// NewChapterSorter initializes the sorter with the strings to analyze.
func NewChapterSorter(strs []string) *ChapterSorter {
	keys := make(map[string]*KeyRange)

	for _, str := range strs {
		for k, v := range scan(str) {
			if kr, ok := keys[k]; ok {
				kr.Update(v)
			} else {
				keys[k] = NewKeyRange(v)
			}
		}
	}

	// Select keys present in over half of the strings
	topKeys := []string{}
	half := len(strs) / 2
	for k, kr := range keys {
		if kr.Count >= half {
			topKeys = append(topKeys, k)
		}
	}

	cs := &ChapterSorter{}
	cs.mergeRepeatedKeyRanges(topKeys, keys)
	cs.sortedKeys = topKeys

	// Sort keys by count desc, then by range desc
	sort.Slice(cs.sortedKeys, func(i, j int) bool {
		a := keys[cs.sortedKeys[i]]
		b := keys[cs.sortedKeys[j]]

		if a.Count != b.Count {
			return b.Count < a.Count
		}
		// Compare range (descending)
		return b.Range().Cmp(a.Range()) > 0
	})

	return cs
}

// keyKind classifies a sanitized scan key. 0 = not used for vol/ch merge, 1 = volume, 2 = chapter.
// We use strict prefix rules so words like "school" (containing the letters "c","h" but not as a
// "Vol"/"Ch" label) are never merged with chapter keys.
func keyKind(san string) int {
	switch {
	case len(san) == 0:
		return 0
	case san == "v" || strings.HasPrefix(san, "vol"):
		return 1
	case san == "ch" || strings.HasPrefix(san, "chap"):
		return 2
	default:
		return 0
	}
}

// mapToTopKey maps a raw scan key onto one of the selected top keys when they denote the
// same field (volume vs chapter), using keyKind. No substring fuzzy matching.
func (cs *ChapterSorter) mapToTopKey(key string) (string, bool) {
	if cs == nil {
		return "", false
	}
	san := sanitizeKey(key)
	if kind := keyKind(san); kind != 0 {
		for _, tk := range cs.sortedKeys {
			if keyKind(sanitizeKey(tk)) == kind {
				return tk, true
			}
		}
	}
	for _, tk := range cs.sortedKeys {
		if key == tk {
			return tk, true
		}
	}
	return key, false
}

// mergeRepeatedKeyRanges merges key ranges of variant keys.
func (cs *ChapterSorter) mergeRepeatedKeyRanges(topKeys []string, keyRanges map[string]*KeyRange) {
	topKeySet := make(map[string]struct{})
	for _, k := range topKeys {
		topKeySet[k] = struct{}{}
	}

	for key, kr := range keyRanges {
		if _, ok := topKeySet[key]; ok {
			continue
		}
		san := sanitizeKey(key)
		kind := keyKind(san)
		if kind == 0 {
			continue
		}
		for _, tk := range topKeys {
			if keyKind(sanitizeKey(tk)) == kind {
				topKR := keyRanges[tk]
				topKR.Update(kr.Min)
				topKR.Update(kr.Max)
				break
			}
		}
	}
}

// Compare compares two chapter strings a and b.
func (cs *ChapterSorter) Compare(a, b string) int {
	itemA := cs.strToItem(a)
	itemB := cs.strToItem(b)
	return itemA.Compare(itemB, cs.sortedKeys)
}

// strToItem parses a string into an Item.
func (cs *ChapterSorter) strToItem(str string) *Item {
	numbers := make(map[string]*big.Float)
	for _, pair := range scanOrdered(str) {
		sanitizedK, found := cs.mapToTopKey(pair.key)
		if !found {
			sanitizedK = pair.key
		}
		numbers[sanitizedK] = pair.value
	}
	return NewItem(numbers)
}

// sanitizeKey removes non-letter characters and lowercases the string.
func sanitizeKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

type scanPair struct {
	key   string
	value *big.Float
}

// scanOrdered returns key/value matches in left-to-right string order. Later matches with
// the same raw key overwrite the map built in scan (last match wins for duplicate keys).
func scanOrdered(str string) []scanPair {
	matches := scanRe.FindAllStringSubmatch(str, -1)
	out := make([]scanPair, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		key := match[1]
		numStr := match[2]
		num, _, err := big.ParseFloat(numStr, 10, 256, big.ToNearestEven)
		if err != nil {
			continue
		}
		out = append(out, scanPair{key: key, value: num})
	}
	return out
}

// scan extracts keys and numeric values from a string.
// If the same key appears more than once, the rightmost (last) match in the string wins.
func scan(str string) map[string]*big.Float {
	pairs := scanOrdered(str)
	result := make(map[string]*big.Float, len(pairs))
	for _, p := range pairs {
		result[p.key] = p.value
	}
	return result
}
