// Package huffman implements an n-ary Huffman coding algorithm.
//
// It can be used to generate prefix-free labels for any number of items.
// Prefix-free labels are labels where for any two labels X and Y,
// there's a guarantee that X is not a prefix of Y.
// This is a useful property because it allows for unambiguous matching.
// When receiving incremental input (e.g. keystrokes),
// as soon as the input matches a prefix-free label X,
// you can stop and process the item corresponding to X.
package huffman

import (
	"container/heap"
)

// Label generates unique prefix-free labels for items given their frequencies.
//
// base is the number of symbols in the alphabet.
// For example, for a binary alphabet, base should be 2.
// base=26 is a common choice for alphabetic labels.
// The mapping from base to alphabet is the caller's responsibility.
// base must be at least 2.
//
// For len(freqs) items, freqs[i] specifies the frequency of item i.
// Items with higher frequencies will be assigned shorter labels.
//
// The returned value is a list of labels for each item.
// labels[i] is the label for item i,
// specified as indexes into the alphabet.
// For example, given a binary alphabet {a b},
// the label {0 1 0} means "aba".
func Label(base int, freqs []int) (labels [][]int) {
	// This implements Huffman coding with a priority queue
	// using the method outlined on Wikipedia [1],
	// altering it for n-ary trees also using advice from the same page.
	// Further advice on optizing comes from rdbliss/huffman [2].
	//
	// [1]: https://en.wikipedia.org/wiki/Huffman_coding#Basic_technique
	// [2]: https://github.com/rdbliss/huffman/

	if base < 2 {
		panic("alphabet must have at least two elements")
	}

	switch len(freqs) {
	case 0:
		return nil
	case 1:
		// special-case:
		// If there's only one item, create a single letter label.
		return [][]int{{0}}
	}

	// Fill the heap with leaf nodes for the user-provided elements.
	nodeHeap := make(nodeHeap, len(freqs))
	for i, f := range freqs {
		nodeHeap[i] = &node{Index: i, Freq: f}
	}
	heap.Init(&nodeHeap)

	// This is the meat of the logic.
	//
	//  - Assign letters [0, $base) to the least frequent items
	//    and remove them from the heap.
	//  - Create a new node that represents these $base items
	//    and push it back into the heap.
	//  - Repeat until there's only one node left in the heap.
	//
	// We'll end up with a tree where each node has up to $base children.
	// The path from root down to leaf nodes will is the label for that item.
	combine := func(numChildren int) {
		children := make([]*node, 0, numChildren)
		var freq int
		for i := 0; i < numChildren && len(nodeHeap) > 0; i++ {
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

	// Special-case: for the first iteration,
	// assign fewer letters to the least frequent items.
	// This will ensure high frequency nodes don't unnecessarily
	// get longer labels because a couple extra nodes pushed them
	// over the edge of $base, requiring a new branch above.
	//
	// See https://github.com/rdbliss/huffman/blob/master/notes.md#generalization
	initial := 2 + (len(nodeHeap)-2)%(base-1)
	if initial > 0 {
		combine(initial)
	}

	for len(nodeHeap) > 1 {
		combine(base)
	}

	// nodeHeap now contains a single element. We'll use it now as a stack
	// to iterate through the node tree.
	labels = make([][]int, len(freqs))

	var labelNode func(*node, []int)
	labelNode = func(n *node, prefix []int) {
		// If we found a leaf, copy the label
		// (iterating into sibilings will mutate it).
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

	// Up to base children of the node. This is nil for leaf nodes.
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
