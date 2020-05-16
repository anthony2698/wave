package telesync

import (
	"strconv"
	"strings"
)

// Buf represents a generic dictionary-like buffer.
type Buf interface {
	// get cursor at key k
	get(k string) (Cur, bool)
	// overwrite all records
	put(v interface{})
	// set record at key k
	set(k string, v interface{})
	// dump contents
	dump() interface{}
}

// Card represents an item on a Page, and holds attributes and data for rendering views.
type Card struct {
	data map[string]interface{}
}

func newBuf(t Typ, n int) Buf {
	if len(t.f) > 0 {
		if n > 0 {
			return newFixBuf(t, n)
		} else if n < 0 {
			return newCycBuf(t, -n, 0)
		} else {
			return newMapBuf(t)
		}
	}
	return nil
}

func debuf(ns *Namespace, s interface{}) Buf {
	if spec, ok := s.(string); ok {
		// fix: "10 foo bar baz"
		// var: "0 foo bar baz"
		// cyc: "-25 foo bar baz"
		fields := strings.Fields(spec)
		if len(fields) > 1 {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				return newBuf(ns.make(fields[1:]), n)
			}
		}
	}
	return nil
}

func newCard(ns *Namespace, ix interface{}) *Card {
	card := &Card{make(map[string]interface{})}
	if x, ok := ix.(map[string]interface{}); ok {
		ks := make([]string, 1)
		for k, v := range x {
			if len(k) > 0 && strings.HasPrefix(k, "#") {
				if b := debuf(ns, v); b != nil {
					k, v = strings.TrimPrefix(k, "#"), b
				}
			}
			ks[0] = k
			card.set(ks, v)
		}
	}
	return card
}

func (c *Card) set(ks []string, v interface{}) {
	switch len(ks) {
	case 0: // should not get here; outer interpreter loop will clobber this card.
		return
	case 1:
		p := ks[0]
		if ib, ok := c.data[p]; ok { // TODO can optimize by duplicating all bufs in a card.bufs map
			if b, ok := ib.(Buf); ok { // avoid clobbering buffers; overwrite instead.
				b.put(v)
				return
			}
		}
		if v == nil {
			delete(c.data, p)
		} else {
			c.data[p] = v
		}
	default:
		var x interface{} = c.data
		p := ks[len(ks)-1]
		for _, k := range ks[:len(ks)-1] {
			x = get(x, k)
		}
		set(x, p, v)
	}
}

func (c *Card) dump() CardD {
	m := make(map[string]interface{})
	for k, iv := range c.data {
		if v, ok := iv.(Buf); ok {
			m["#"+k] = v.dump()
		} else {
			m[k] = dump(iv)
		}
	}
	return CardD{m}
}

func dump(ix interface{}) interface{} {
	switch x := ix.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			m[k] = dump(v)
		}
		return m
	case []interface{}:
		s := make([]interface{}, len(x))
		for i, v := range x {
			s[i] = dump(v)
		}
	}
	return ix
}

func set(ix interface{}, k string, v interface{}) {
	switch x := ix.(type) {
	case Buf:
		x.set(k, v)
	case Cur:
		x.set(k, v)
	case map[string]interface{}:
		if v == nil {
			delete(x, k)
		} else {
			x[k] = v
		}
	case []interface{}:
		if i, err := strconv.Atoi(k); err == nil {
			if i >= 0 && i < len(x) {
				x[i] = v
			}
		}
	}
}

func get(ix interface{}, k string) interface{} {
	switch x := ix.(type) {
	case Buf:
		if r, ok := x.get(k); ok {
			return r
		}
	case Cur:
		return x.get(k)
	case map[string]interface{}:
		if v, ok := x[k]; ok {
			return v
		}
	case []interface{}:
		if i, err := strconv.Atoi(k); err == nil {
			if i >= 0 && i < len(x) {
				return x[i]
			}
		}
	}
	return nil
}
