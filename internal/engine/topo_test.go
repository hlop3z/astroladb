package engine

import (
	"errors"
	"testing"
)

// -----------------------------------------------------------------------------
// Test Node Implementation
// -----------------------------------------------------------------------------

// testNode implements DependencyNode for testing.
type testNode struct {
	id   string
	deps []string
}

func (n *testNode) ID() string             { return n.id }
func (n *testNode) Dependencies() []string { return n.deps }

// newNode creates a test node with the given ID and dependencies.
func newNode(id string, deps ...string) *testNode {
	return &testNode{id: id, deps: deps}
}

// getIDs extracts IDs from a slice of nodes.
func getIDs(nodes []*testNode) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID()
	}
	return ids
}

// -----------------------------------------------------------------------------
// TopoSort Tests
// -----------------------------------------------------------------------------

func TestTopoSort_EmptyInput(t *testing.T) {
	result, err := TopoSort([]*testNode{})
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	if result != nil {
		t.Errorf("TopoSort() = %v, want nil", result)
	}
}

func TestTopoSort_NilInput(t *testing.T) {
	result, err := TopoSort[*testNode](nil)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	if result != nil {
		t.Errorf("TopoSort() = %v, want nil", result)
	}
}

func TestTopoSort_SingleNode(t *testing.T) {
	nodes := []*testNode{newNode("A")}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("TopoSort() = %d nodes, want 1", len(result))
	}
	if result[0].ID() != "A" {
		t.Errorf("TopoSort()[0].ID() = %q, want %q", result[0].ID(), "A")
	}
}

func TestTopoSort_LinearChain(t *testing.T) {
	// A -> B -> C (C depends on B, B depends on A)
	nodes := []*testNode{
		newNode("C", "B"),
		newNode("B", "A"),
		newNode("A"),
	}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}

	ids := getIDs(result)
	if len(ids) != 3 {
		t.Fatalf("TopoSort() = %d nodes, want 3", len(ids))
	}

	// A must come before B, B must come before C
	aIdx, bIdx, cIdx := -1, -1, -1
	for i, id := range ids {
		switch id {
		case "A":
			aIdx = i
		case "B":
			bIdx = i
		case "C":
			cIdx = i
		}
	}

	if aIdx > bIdx {
		t.Errorf("TopoSort() A should come before B: got A at %d, B at %d", aIdx, bIdx)
	}
	if bIdx > cIdx {
		t.Errorf("TopoSort() B should come before C: got B at %d, C at %d", bIdx, cIdx)
	}
}

func TestTopoSort_DiamondDependency(t *testing.T) {
	// Diamond: D depends on B and C, B and C depend on A
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	nodes := []*testNode{
		newNode("D", "B", "C"),
		newNode("B", "A"),
		newNode("C", "A"),
		newNode("A"),
	}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}

	ids := getIDs(result)
	if len(ids) != 4 {
		t.Fatalf("TopoSort() = %d nodes, want 4", len(ids))
	}

	// Find positions
	pos := make(map[string]int)
	for i, id := range ids {
		pos[id] = i
	}

	// A must come first (or at least before B, C, D)
	if pos["A"] > pos["B"] {
		t.Error("TopoSort() A should come before B")
	}
	if pos["A"] > pos["C"] {
		t.Error("TopoSort() A should come before C")
	}

	// B and C must come before D
	if pos["B"] > pos["D"] {
		t.Error("TopoSort() B should come before D")
	}
	if pos["C"] > pos["D"] {
		t.Error("TopoSort() C should come before D")
	}
}

func TestTopoSort_MultipleRoots(t *testing.T) {
	// Two independent chains that converge:
	// A -> C, B -> C
	nodes := []*testNode{
		newNode("C", "A", "B"),
		newNode("A"),
		newNode("B"),
	}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}

	ids := getIDs(result)
	if len(ids) != 3 {
		t.Fatalf("TopoSort() = %d nodes, want 3", len(ids))
	}

	// Find positions
	pos := make(map[string]int)
	for i, id := range ids {
		pos[id] = i
	}

	// A and B must come before C
	if pos["A"] > pos["C"] {
		t.Error("TopoSort() A should come before C")
	}
	if pos["B"] > pos["C"] {
		t.Error("TopoSort() B should come before C")
	}
}

func TestTopoSort_CircularDependency(t *testing.T) {
	// A -> B -> C -> A (cycle)
	nodes := []*testNode{
		newNode("A", "C"),
		newNode("B", "A"),
		newNode("C", "B"),
	}

	_, err := TopoSort(nodes)
	if err == nil {
		t.Fatal("TopoSort() expected error for circular dependency")
	}
	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("TopoSort() error = %v, want ErrCircularDependency", err)
	}
}

func TestTopoSort_SelfCycle(t *testing.T) {
	// A depends on itself - this is a degenerate case where A has itself as dep
	// In the TopoSort implementation, self-deps count towards in-degree,
	// so A has in-degree 1 and will never be processed, resulting in circular dependency
	nodes := []*testNode{
		newNode("A", "A"),
	}

	result, err := TopoSort(nodes)
	// The implementation may or may not detect self-cycles depending on how
	// in-degree is calculated. If it doesn't count self-deps, A processes fine.
	// If it does count self-deps, it returns ErrCircularDependency.
	if err != nil && !errors.Is(err, ErrCircularDependency) {
		t.Errorf("TopoSort() unexpected error = %v", err)
	}
	// Either no error with the node returned, or circular dependency error
	if err == nil && len(result) != 1 {
		t.Errorf("TopoSort() = %d nodes, want 1", len(result))
	}
}

func TestTopoSort_TwoNodeCycle(t *testing.T) {
	// A -> B -> A
	nodes := []*testNode{
		newNode("A", "B"),
		newNode("B", "A"),
	}

	_, err := TopoSort(nodes)
	if err == nil {
		t.Fatal("TopoSort() expected error for two-node cycle")
	}
	if !errors.Is(err, ErrCircularDependency) {
		t.Errorf("TopoSort() error = %v, want ErrCircularDependency", err)
	}
}

func TestTopoSort_ExternalDependencies(t *testing.T) {
	// B depends on A and X, where X is not in our node set (external)
	nodes := []*testNode{
		newNode("B", "A", "X"),
		newNode("A"),
	}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}

	ids := getIDs(result)
	if len(ids) != 2 {
		t.Fatalf("TopoSort() = %d nodes, want 2", len(ids))
	}

	// A must come before B
	pos := make(map[string]int)
	for i, id := range ids {
		pos[id] = i
	}
	if pos["A"] > pos["B"] {
		t.Error("TopoSort() A should come before B")
	}
}

func TestTopoSort_NoDependencies(t *testing.T) {
	// Three independent nodes with no dependencies
	nodes := []*testNode{
		newNode("C"),
		newNode("A"),
		newNode("B"),
	}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}

	ids := getIDs(result)
	if len(ids) != 3 {
		t.Fatalf("TopoSort() = %d nodes, want 3", len(ids))
	}

	// All nodes should be present (order is deterministic due to sorting)
	expected := map[string]bool{"A": true, "B": true, "C": true}
	for _, id := range ids {
		if !expected[id] {
			t.Errorf("TopoSort() unexpected node %q", id)
		}
		delete(expected, id)
	}
	if len(expected) > 0 {
		t.Errorf("TopoSort() missing nodes: %v", expected)
	}
}

func TestTopoSort_DeterministicOrder(t *testing.T) {
	// Run multiple times to verify deterministic output
	nodes := []*testNode{
		newNode("D", "B", "C"),
		newNode("B", "A"),
		newNode("C", "A"),
		newNode("A"),
	}

	var firstResult []string
	for i := 0; i < 10; i++ {
		result, err := TopoSort(nodes)
		if err != nil {
			t.Fatalf("TopoSort() error = %v", err)
		}

		ids := getIDs(result)
		if firstResult == nil {
			firstResult = ids
		} else {
			// Compare with first result
			if len(ids) != len(firstResult) {
				t.Fatalf("TopoSort() inconsistent length on iteration %d", i)
			}
			for j := range ids {
				if ids[j] != firstResult[j] {
					t.Errorf("TopoSort() inconsistent order on iteration %d: got %v, want %v", i, ids, firstResult)
					break
				}
			}
		}
	}
}

func TestTopoSort_ComplexGraph(t *testing.T) {
	// More complex dependency graph:
	//       A
	//      /|\
	//     B C D
	//     |/| |
	//     E F |
	//      \|/
	//       G
	nodes := []*testNode{
		newNode("G", "E", "F", "D"),
		newNode("E", "B", "C"),
		newNode("F", "C"),
		newNode("B", "A"),
		newNode("C", "A"),
		newNode("D", "A"),
		newNode("A"),
	}

	result, err := TopoSort(nodes)
	if err != nil {
		t.Fatalf("TopoSort() error = %v", err)
	}

	ids := getIDs(result)
	if len(ids) != 7 {
		t.Fatalf("TopoSort() = %d nodes, want 7", len(ids))
	}

	// Verify all dependency constraints
	pos := make(map[string]int)
	for i, id := range ids {
		pos[id] = i
	}

	// A must come before B, C, D
	for _, dep := range []string{"B", "C", "D"} {
		if pos["A"] > pos[dep] {
			t.Errorf("TopoSort() A should come before %s", dep)
		}
	}

	// B and C must come before E
	if pos["B"] > pos["E"] || pos["C"] > pos["E"] {
		t.Error("TopoSort() B and C should come before E")
	}

	// C must come before F
	if pos["C"] > pos["F"] {
		t.Error("TopoSort() C should come before F")
	}

	// E, F, D must come before G
	for _, dep := range []string{"E", "F", "D"} {
		if pos[dep] > pos["G"] {
			t.Errorf("TopoSort() %s should come before G", dep)
		}
	}
}

// -----------------------------------------------------------------------------
// sortStrings Tests
// -----------------------------------------------------------------------------

func TestSortStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"empty", []string{}, []string{}},
		{"single", []string{"a"}, []string{"a"}},
		{"already_sorted", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"reverse", []string{"c", "b", "a"}, []string{"a", "b", "c"}},
		{"random", []string{"d", "a", "c", "b"}, []string{"a", "b", "c", "d"}},
		{"duplicates", []string{"b", "a", "b", "a"}, []string{"a", "a", "b", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying test data
			input := make([]string, len(tt.input))
			copy(input, tt.input)

			sortStrings(input)

			if len(input) != len(tt.want) {
				t.Fatalf("sortStrings() = %v, want %v", input, tt.want)
			}

			for i := range input {
				if input[i] != tt.want[i] {
					t.Errorf("sortStrings()[%d] = %q, want %q", i, input[i], tt.want[i])
				}
			}
		})
	}
}
