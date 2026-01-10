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
