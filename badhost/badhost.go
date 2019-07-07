package badhost

import (
	"log"
	"time"

	"github.com/derekparker/trie"
	"github.com/lvht/tlstunnel/badhost/gfwlist"
)

// Pool 保存无法直连的主机列表
type Pool struct {
	black *trie.Trie
	temp  *trie.Trie
}

// NewPool creat new Pool
func NewPool(useGFWList bool) (p *Pool, err error) {
	p = &Pool{
		black: trie.New(),
		temp:  trie.New(),
	}

	if useGFWList {
		go p.loadGFWList()
	}

	return
}

func (p *Pool) loadGFWList() {
	log.Println("start loading gfwlist")
	domains, err := gfwlist.FetchBlockedDomains()
	if err != nil {
		return
	}

	for _, d := range domains {
		d = strrev(d)
		p.black.Add(d, nil)
	}
	log.Println("gfwlist loaded")
}

// Add 添加一个主机地址，如 google.com
func (p *Pool) Add(host string) {
	host = strrev(host)
	p.temp.Add(host, time.Now())
}

// HasSuffix 判断是否有无法连接的记录
func (p *Pool) HasSuffix(host string) bool {
	host = strrev(host)

	if p.temp.HasKeysWithPrefix(host) {
		return true
	}

	if p.black.HasKeysWithPrefix(host) {
		return true
	}

	return false
}

func strrev(s string) string {
	l := len(s)
	if l == 0 {
		return s
	}

	rs := make([]byte, l)

	for i := l - 1; i >= 0; i-- {
		rs[l-1-i] = s[i]
	}

	return string(rs)
}
