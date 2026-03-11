package stdlib

import (
	"encoding/json"
	"fmt"
	"iter"
	"reflect"
	"sort"
)

// OrderedMap preserves insertion order of keys, matching Ruby's Hash semantics.
type OrderedMap[K comparable, V any] struct {
	Data map[K]V
	keys []K
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{Data: map[K]V{}}
}

func (m *OrderedMap[K, V]) Set(key K, val V) {
	if _, exists := m.Data[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.Data[key] = val
}

func (m *OrderedMap[K, V]) Delete(key K) {
	if _, exists := m.Data[key]; exists {
		delete(m.Data, key)
		for i, k := range m.keys {
			if k == key {
				m.keys = append(m.keys[:i], m.keys[i+1:]...)
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) Len() int {
	return len(m.Data)
}

// MarshalJSON serializes the OrderedMap to JSON, preserving insertion order.
func (m *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	buf := []byte{'{'}
	for i, k := range m.keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		valBytes, err := json.Marshal(m.Data[k])
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')
		buf = append(buf, valBytes...)
	}
	buf = append(buf, '}')
	return buf, nil
}

func (m *OrderedMap[K, V]) Keys() []K {
	result := make([]K, len(m.keys))
	copy(result, m.keys)
	return result
}

func (m *OrderedMap[K, V]) Values() []V {
	result := make([]V, 0, len(m.keys))
	for _, k := range m.keys {
		result = append(result, m.Data[k])
	}
	return result
}

func (m *OrderedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range m.keys {
			if !yield(k, m.Data[k]) {
				return
			}
		}
	}
}

func (m *OrderedMap[K, V]) Clear() {
	m.Data = map[K]V{}
	m.keys = nil
}

func (m *OrderedMap[K, V]) HasKey(key K) bool {
	_, ok := m.Data[key]
	return ok
}

func (m *OrderedMap[K, V]) HasValue(val V) bool {
	for _, v := range m.Data {
		if reflect.DeepEqual(v, val) {
			return true
		}
	}
	return false
}

func OrderedMapMerge[K comparable, V any](base, other *OrderedMap[K, V]) *OrderedMap[K, V] {
	merged := NewOrderedMap[K, V]()
	for _, k := range base.keys {
		merged.Set(k, base.Data[k])
	}
	for _, k := range other.keys {
		merged.Set(k, other.Data[k])
	}
	return merged
}

func OrderedMapKey[K comparable, V comparable](m *OrderedMap[K, V], val V) K {
	for _, k := range m.keys {
		if m.Data[k] == val {
			return k
		}
	}
	var zero K
	return zero
}

func OrderedMapMergeBlock[K comparable, V any](base, other *OrderedMap[K, V], fn func(K, V, V) V) *OrderedMap[K, V] {
	merged := NewOrderedMap[K, V]()
	for _, k := range base.keys {
		merged.Set(k, base.Data[k])
	}
	for _, k := range other.keys {
		if _, exists := base.Data[k]; exists {
			merged.Set(k, fn(k, base.Data[k], other.Data[k]))
		} else {
			merged.Set(k, other.Data[k])
		}
	}
	return merged
}

func OrderedMapInvert[K comparable, V comparable](m *OrderedMap[K, V]) *OrderedMap[V, K] {
	inverted := NewOrderedMap[V, K]()
	for _, k := range m.keys {
		inverted.Set(m.Data[k], k)
	}
	return inverted
}

func OrderedMapShift[K comparable, V any](m *OrderedMap[K, V]) (K, V) {
	if len(m.keys) == 0 {
		var zk K
		var zv V
		return zk, zv
	}
	k := m.keys[0]
	v := m.Data[k]
	m.Delete(k)
	return k, v
}

func OrderedMapToA[K comparable, V any](m *OrderedMap[K, V]) [][]any {
	result := make([][]any, 0, len(m.keys))
	for _, k := range m.keys {
		result = append(result, []any{k, m.Data[k]})
	}
	return result
}

// NewOrderedMapFromGoMap creates an OrderedMap from a native Go map.
// Used as a bridge when Go library code returns map[K]V but thanos
// expects *OrderedMap[K,V]. Keys are sorted for deterministic order.
func NewOrderedMapFromGoMap[K comparable, V any](m map[K]V) *OrderedMap[K, V] {
	om := NewOrderedMap[K, V]()
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%v", keys[i]) < fmt.Sprintf("%v", keys[j])
	})
	for _, k := range keys {
		om.Set(k, m[k])
	}
	return om
}

func OrderedMapSortBy[K comparable, V any, R Ordered](m *OrderedMap[K, V], fn func(K, V) R) *OrderedMap[K, V] {
	type entry struct {
		key K
		val V
		by  R
	}
	entries := make([]entry, 0, len(m.keys))
	for _, k := range m.keys {
		entries = append(entries, entry{k, m.Data[k], fn(k, m.Data[k])})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].by < entries[j].by
	})
	result := NewOrderedMap[K, V]()
	for _, e := range entries {
		result.Set(e.key, e.val)
	}
	return result
}

func OrderedMapMinBy[K comparable, V any, R Ordered](m *OrderedMap[K, V], fn func(K, V) R) (K, V) {
	var bestK K
	var bestV V
	var bestR R
	first := true
	for _, k := range m.keys {
		v := m.Data[k]
		r := fn(k, v)
		if first || r < bestR {
			bestK, bestV, bestR = k, v, r
			first = false
		}
	}
	return bestK, bestV
}

func OrderedMapMaxBy[K comparable, V any, R Ordered](m *OrderedMap[K, V], fn func(K, V) R) (K, V) {
	var bestK K
	var bestV V
	var bestR R
	first := true
	for _, k := range m.keys {
		v := m.Data[k]
		r := fn(k, v)
		if first || r > bestR {
			bestK, bestV, bestR = k, v, r
			first = false
		}
	}
	return bestK, bestV
}
