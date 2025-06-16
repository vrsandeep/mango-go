package util

import (
	"math/big"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

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

// hasKeyWithVariation checks if key is a variant of any available key.
func (cs *ChapterSorter) hasKeyWithVariation(key string, availableKeys []string) (string, bool) {
	sanitizedKey := sanitizeKey(key)
	for _, k := range availableKeys {
		sanitizedK := sanitizeKey(k)
		if strings.Contains(sanitizedKey, sanitizedK) || strings.Contains(sanitizedK, sanitizedKey) {
			return k, true
		}
	}
	return "", false
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
		if topKey, found := cs.hasKeyWithVariation(key, topKeys); found {
			topKR := keyRanges[topKey]
			topKR.Update(kr.Min)
			topKR.Update(kr.Max)
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
	for k, v := range scan(str) {
		sanitizedK, found := cs.hasKeyWithVariation(k, cs.sortedKeys)
		if !found {
			sanitizedK = k
		}
		numbers[sanitizedK] = v
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

// scan extracts keys and numeric values from a string.
// Returns a map of key -> *big.Float
func scan(str string) map[string]*big.Float {
	// Regex to match: ([^0-9\n\r ]*)[ ]*([0-9]*\.?[0-9]+)
	// i.e. key (non-digit chars), optional spaces, then number (int or float)
	re := regexp.MustCompile(`([^0-9\n\r ]*)[ ]*([0-9]*\.?[0-9]+)`)
	matches := re.FindAllStringSubmatch(str, -1)

	result := make(map[string]*big.Float)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		key := match[1]
		numStr := match[2]

		num, _, err := big.ParseFloat(numStr, 10, 256, big.ToNearestEven)
		if err != nil {
			// Ignore parse errors, or optionally handle them
			continue
		}
		result[key] = num
	}
	return result
}
