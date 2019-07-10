package badhost

import (
	"log"

	"github.com/lvht/tlstunnel/badhost/gfwlist"
)

// Pool 保存无法直连的主机列表
type Pool struct {
	black *Trie
	temp  *Trie
}

// NewPool creat new Pool
func NewPool(useGFWList bool) (p *Pool, err error) {
	p = &Pool{
		black: NewTrie(),
		temp:  NewTrie(),
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
		p.black.Add(d)
	}
	log.Println("gfwlist loaded")
}

// Add 添加一个主机地址，如 google.com
func (p *Pool) Add(host string) {
	host = strrev(host)
	p.temp.Add(host)
}

// HasSuffix 判断是否有无法连接的记录
func (p *Pool) HasSuffix(host string) bool {
	host = strrev(host)

	if p.temp.Find(host) {
		return true
	}

	if p.black.Find(host) {
		return true
	}

	return false
}

// abc -> cba
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
