// Package huffman implements an n-ary Huffman coding algorithm
// to generate prefix-free labels for a set of items.
package huffman

import "container/heap"

// Label generates unique prefix-free labels for a set of items given their
// frequencies. Prefix-free labels guarantee that none of the generated labels
// is a prefix for another.
//
// For each item i, freqs[i] specifies its frequency. Items with higher
// frequencies will get shorter labels.
//
// Labels are generated using an alphabet of the provided size. For each item
// i, labels[i] is a list of indexes in the alphabet that comprise that item's
// label. For example, given a binary alphabet {a b}, the label {0 1 0} means
// "aba".
func Label(alphabetSize int, freqs []int) (labels [][]int) {
	// This implements Huffman coding using a priority queue using the
	// method outlined on Wikipedia [1], altering it for n-ary trees also
	// using advice from the same page. In the interest of keeping the
	// solution simple, we don't generate low-frequency placeholder nodes.
	//
	// [1]: https://en.wikipedia.org/wiki/Huffman_coding#Basic_technique

	if alphabetSize < 2 {
		panic("alphabet must have at least two elements")
	}

	switch len(freqs) {
	case 0:
		return nil
	case 1:
		// special-case: If there's only one item, create a single
		// letter alabel.
		return [][]int{{0}}
	}

	// Fill the heap with leaf nodes for the user-provided elements.
	nodeHeap := make(nodeHeap, len(freqs))
	for i, f := range freqs {
		nodeHeap[i] = &node{Index: i, Freq: f}
	}
	heap.Init(&nodeHeap)

	for len(nodeHeap) > 1 {
		var (
			children []*node
			freq     int
		)
		for i := 0; i < alphabetSize && len(nodeHeap) > 0; i++ {
			child := heap.Pop(&nodeHeap).(*node)
			children = append(children, child)
			freq += child.Freq
		}

		heap.Push(&nodeHeap, &node{
			Index:    -1,
			Children: children,
			Freq:     freq,
		})
	}

	// nodeHeap now contains a single element. We'll use it now as a stack
	// to iterate through the node tree.
	labels = make([][]int, len(freqs))

	var labelNode func(*node, []int)
	labelNode = func(n *node, prefix []int) {
		// If we found a leaf, copy the label (iterating into sibilings
		// will mutate it).
		if i := n.Index; i >= 0 {
			label := make([]int, len(prefix))
			copy(label, prefix)
			labels[i] = label
			return
		}

		// For branches, iterate through their children, prefixing the
		// corresponding label rune.
		for i, c := range n.Children {
			labelNode(c, append(prefix, i))
		}
	}

	labelNode(heap.Pop(&nodeHeap).(*node), nil)
	return labels
}

type node struct {
	// Index of the leaf node, as identified by the user. This is -1 for
	// branch nodes.
	Index int

	// Up to alphabetSize children of the node. This is nil for leaf nodes.
	Children []*node

	// Frequency of the leaf node, or the combined frequency of the leaf
	// nodes of a branch node.
	Freq int
}

type nodeHeap []*node

func (ns nodeHeap) Len() int { return len(ns) }

func (ns nodeHeap) Less(i, j int) bool {
	return ns[i].Freq < ns[j].Freq
}

func (ns nodeHeap) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

func (ns *nodeHeap) Push(e interface{}) {
	*ns = append(*ns, e.(*node))
}

func (ns *nodeHeap) Pop() interface{} {
	n := len(*ns) - 1
	v := (*ns)[n]
	*ns = (*ns)[:n]
	return v
}
