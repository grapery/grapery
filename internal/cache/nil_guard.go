package cache

import "reflect"

// IsEffectivelyNil reports whether c is unusable for Redis-backed operations:
// untyped nil, or an interface value holding a nil pointer (Go "typed nil").
func IsEffectivelyNil(c Cache) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}
