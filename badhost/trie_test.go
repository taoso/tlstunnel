package badhost

import (
	"testing"
)

func TestFind(t *testing.T) {
	trie := NewTrie()

	trie.Add("com.baidu")

	if !trie.Find("com.baidu") || !trie.Find("com.baidu.www") {
		t.Fatal("Could not find node")
	}
}
