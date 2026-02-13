// Package engine provides migration planning, diffing, and execution.
package engine

// ToMap converts a slice to a map using a key function.
// Example: ToMap(users, func(u User) string { return u.ID })
func ToMap[T any, K comparable](items []T, key func(T) K) map[K]T {
	m := make(map[K]T, len(items))
	for _, item := range items {
		m[key(item)] = item
	}
	return m
}

// ToSet converts a slice to a set (map with empty struct values).
// Example: ToSet([]string{"a", "b", "c"})
func ToSet[T comparable](items []T) map[T]struct{} {
	m := make(map[T]struct{}, len(items))
	for _, item := range items {
		m[item] = struct{}{}
	}
	return m
}

// ParseSimpleRef parses a simple "ns.table" or "table" reference.
func ParseSimpleRef(ref string) (ns, table string, isRelative bool) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '.' {
			if i == 0 {
				// ".table" - relative
				return "", ref[1:], true
			}
			return ref[:i], ref[i+1:], false
		}
	}
	// No dot - just table name
	return "", ref, false
}
