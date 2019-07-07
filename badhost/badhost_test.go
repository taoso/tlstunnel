package badhost

import "testing"

func TestPool(t *testing.T) {
	p, _ := NewPool(false)
	p.Add("g.cn")

	if !p.HasSuffix("g.cn") {
		t.Error("invalid HasSuffix")
	}

	if p.HasSuffix("z.cn") {
		t.Error("invalid HasSuffix")
	}

	if p.HasSuffix("z.g.cn") {
		t.Error("invalid HasSuffix")
	}
}
