package gorpm

import (
	"fmt"
	"reflect"
)

func Compare(x, y interface{}) (bool, []string) {
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return false, []string{"root type not consistent"}
	}
	reason := &struct{ List []string }{List: make([]string, 0)}
	ret := deepValueEqual(v1, v2, reason, 0)
	return ret, reason.List
}

func deepValueEqual(v1, v2 reflect.Value, reason *struct{ List []string }, depth int) bool {
	if v1.Type() != v2.Type() {
		return false
	}
	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), reason, depth+1) {
				reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(array) index(%d) inconsistent", depth, i))
				return false
			}
		}
		return true
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(slice) length not equal", depth))
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		for i := 0; i < v1.Len(); i++ {
			if !deepValueEqual(v1.Index(i), v2.Index(i), reason, depth+1) {
				reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(slice) index(%d) inconsistent", depth, i))
				return false
			}
		}
		return true
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() == v2.IsNil()
		}
		if deepValueEqual(v1.Elem(), v2.Elem(), reason, depth+1) {
			return true
		} else {
			reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(interface) inconsistent", depth))
			return false
		}
	case reflect.Ptr:
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		if deepValueEqual(v1.Elem(), v2.Elem(), reason, depth+1) {
			return true
		} else {
			reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(pointer) inconsistent", depth))
			return false
		}
	case reflect.Struct:
		for i, n := 0, v1.NumField(); i < n; i++ {
			if !deepValueEqual(v1.Field(i), v2.Field(i), reason, depth+1) {
				reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(struct) field(%s) inconsistent", depth, v1.Type().Field(i).Name))
				return false
			}
		}
		return true
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.Len() != v2.Len() {
			return false
		}
		if v1.Pointer() == v2.Pointer() {
			return true
		}
		for _, k := range v1.MapKeys() {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() || !val2.IsValid() || !deepValueEqual(val1, val2, reason, depth+1) {
				reason.List = append(reason.List, fmt.Sprintf("depth(%d) type(map) key(%s) inconsistent", depth, k.String()))
				return false
			}
		}
		return true
	default:
		if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
			return false
		}
		return true
	}
}
