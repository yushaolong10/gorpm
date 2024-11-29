package gorpm

import (
	"fmt"
	"reflect"
)

func Compare(x, y interface{}) (bool, *ReasonNode) {
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return false, &ReasonNode{Difference: Difference{Depth: 0, Type: "paramType", Reason: "inconsistent"}}
	}
	rn := &ReasonNode{}
	ret := deepValueEqual(v1, v2, rn, 0)
	return ret, rn
}

func deepValueEqual(v1, v2 reflect.Value, rn *ReasonNode, depth int) bool {
	if v1.Type() != v2.Type() {
		rn.Difference = Difference{
			Depth:  depth,
			Type:   "paramType",
			Reason: "inconsistent",
		}
		return false
	}
	var ok = true
	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			next := &ReasonNode{}
			if !deepValueEqual(v1.Index(i), v2.Index(i), next, depth+1) {
				ok = false
				rn.AddListNode(&ReasonNode{
					Difference: Difference{
						Depth:  depth,
						Type:   "array",
						Reason: fmt.Sprintf("index(%d) value inconsistent", i),
					},
					Next: next,
				})
			}
		}
		if !ok {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "array",
				Reason: "inconsistent",
			}
			return false
		}
		return true
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "slice",
				Reason: "length not equal",
			}
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		for i := 0; i < v1.Len(); i++ {
			next := &ReasonNode{}
			if !deepValueEqual(v1.Index(i), v2.Index(i), next, depth+1) {
				ok = false
				rn.AddListNode(&ReasonNode{
					Difference: Difference{
						Depth:  depth,
						Type:   "slice",
						Reason: fmt.Sprintf("index(%d) value inconsistent", i),
					},
					Next: next,
				})
			}
		}
		if !ok {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "slice",
				Reason: "inconsistent",
			}
			return false
		}
		return true
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() == v2.IsNil()
		}
		next := &ReasonNode{}
		if deepValueEqual(v1.Elem(), v2.Elem(), next, depth+1) {
			return true
		} else {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "interface",
				Reason: fmt.Sprintf("inconsistent"),
			}
			if next.Depth > 0 {
				rn.Next = next
			}
			return false
		}
	case reflect.Ptr:
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		next := &ReasonNode{}
		if deepValueEqual(v1.Elem(), v2.Elem(), next, depth+1) {
			return true
		} else {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "pointer",
				Reason: fmt.Sprintf("inconsistent"),
			}
			rn.Next = next
			return false
		}
	case reflect.Struct:
		for i, n := 0, v1.NumField(); i < n; i++ {
			next := &ReasonNode{}
			if !deepValueEqual(v1.Field(i), v2.Field(i), next, depth+1) {
				ok = false
				rn.AddListNode(&ReasonNode{
					Difference: Difference{
						Depth:  depth,
						Type:   "struct",
						Reason: fmt.Sprintf("field(%s) value inconsistent", v1.Type().Field(i).Name),
					},
					Next: next,
				})
			}
		}
		if !ok {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "struct",
				Reason: fmt.Sprintf("inconsistent"),
			}
			return false
		}
		return true
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		for _, k := range v1.MapKeys() {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() || !val2.IsValid() {
				ok = false
				rn.AddListNode(&ReasonNode{
					Difference: Difference{
						Depth:  depth,
						Type:   "map",
						Reason: fmt.Sprintf("key(%s) not exist", k.String()),
					},
				})
				continue
			}
			next := &ReasonNode{}
			if !deepValueEqual(val1, val2, next, depth+1) {
				ok = false
				rn.AddListNode(&ReasonNode{
					Difference: Difference{
						Depth:  depth,
						Type:   "map",
						Reason: fmt.Sprintf("key(%s) value inconsistent", k.String()),
					},
					Next: next,
				})
			}
		}
		if !ok {
			rn.Difference = Difference{
				Depth:  depth,
				Type:   "map",
				Reason: fmt.Sprintf("inconsistent"),
			}
			return false
		}
		return true
	default:
		if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
			return false
		}
		return true
	}
}

type ReasonNode struct {
	Difference
	Next *ReasonNode   `json:"next,omitempty"`
	List []*ReasonNode `json:"list,omitempty"`
}

type Difference struct {
	Depth  int    `json:"depth"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func (node *ReasonNode) AddListNode(item *ReasonNode) {
	if node.List == nil {
		node.List = make([]*ReasonNode, 0)
	}
	node.List = append(node.List, item)
}
