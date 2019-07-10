package badhost

// https://github.com/derekparker/trie.git

import (
	"sync"
)

var rwm sync.RWMutex

type trieNode struct {
	val      rune
	term     bool
	children map[rune]*trieNode
}

// Trie a R-Way Trie data structure.
type Trie struct {
	root *trieNode
}

const nul = 0x0

// NewTrie creates a new Trie with an initialized root Node.
func NewTrie() *Trie {
	return &Trie{
		root: &trieNode{children: make(map[rune]*trieNode)},
	}
}

// Add adds the key to the Trie.
func (t *Trie) Add(key string) bool {
	if t == nil {
		return false
	}

	rwm.Lock()
	defer rwm.Unlock()

	runes := []rune(key)
	node := t.root
	for i := range runes {
		r := runes[i]
		if n, ok := node.children[r]; ok {
			node = n
		} else {
			node = node.newChild(r, false)
		}
	}
	node.newChild(nul, true)

	return true
}

// Find meta data associated with `key`.
func (t *Trie) Find(key string) bool {
	if t == nil {
		return false
	}

	rwm.RLock()
	defer rwm.RUnlock()

	node := findNode(t.root, []rune(key))
	if node == nil {
		return false
	}

	node, ok := node.children[nul]
	if !ok || !node.term {
		return false
	}

	return true
}

func (n *trieNode) newChild(val rune, term bool) *trieNode {
	node := &trieNode{
		val:      val,
		term:     term,
		children: make(map[rune]*trieNode),
	}
	n.children[val] = node
	return node
}

func findNode(node *trieNode, runes []rune) *trieNode {
	if node == nil {
		return nil
	}

	if len(runes) == 0 {
		return node
	}

	if n, ok := node.children[nul]; ok && n.term {
		return node
	}

	n, ok := node.children[runes[0]]
	if !ok {
		return nil
	}

	var nrunes []rune
	if len(runes) > 1 {
		nrunes = runes[1:]
	} else {
		nrunes = runes[0:0]
	}

	return findNode(n, nrunes)
}
