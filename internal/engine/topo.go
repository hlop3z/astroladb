package engine

import "errors"

// ErrCircularDependency is returned when a circular dependency is detected.
var ErrCircularDependency = errors.New("circular dependency detected")

// DependencyNode represents a node with dependencies for topological sorting.
type DependencyNode interface {
	ID() string
	Dependencies() []string
}

// TopoSort performs topological sort using Kahn's algorithm.
// Returns nodes ordered so that dependencies come before dependents.
// Returns ErrCircularDependency if a cycle is detected.
func TopoSort[T DependencyNode](nodes []T) ([]T, error) {
	if len(nodes) == 0 {
		return nil, nil
	}

	if len(nodes) == 1 {
		return nodes, nil
	}

	// Build node map and collect all node IDs
	nodeMap := make(map[string]T, len(nodes))
	nodeIDs := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		id := n.ID()
		nodeMap[id] = n
		nodeIDs[id] = true
	}

	// Calculate in-degree for each node
	// in-degree = number of dependencies that are in our node set
	inDegree := make(map[string]int, len(nodes))
	for _, n := range nodes {
		id := n.ID()
		count := 0
		for _, dep := range n.Dependencies() {
			// Only count dependencies that are in our set
			if nodeIDs[dep] {
				count++
			}
		}
		inDegree[id] = count
	}

	// Initialize queue with nodes that have no dependencies (in-degree = 0)
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Sort queue for deterministic output
	sortStrings(queue)

	// Process nodes in topological order
	var result []T
	for len(queue) > 0 {
		// Dequeue
		id := queue[0]
		queue = queue[1:]

		if n, ok := nodeMap[id]; ok {
			result = append(result, n)
		}

		// Decrease in-degree for nodes that depend on this one
		for otherId, otherNode := range nodeMap {
			for _, dep := range otherNode.Dependencies() {
				if dep == id {
					inDegree[otherId]--
					if inDegree[otherId] == 0 {
						queue = append(queue, otherId)
						sortStrings(queue) // Keep sorted for determinism
					}
					break
				}
			}
		}
	}

	// Check if we processed all nodes
	if len(result) != len(nodes) {
		return nil, ErrCircularDependency
	}

	return result, nil
}

// sortStrings performs an in-place insertion sort on a small slice.
// Used for maintaining deterministic ordering in the queue.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
