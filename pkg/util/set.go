package util

type Set[T comparable] map[T]struct{}

func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

func (s Set[T]) Add(item T) {
	s[item] = struct{}{}
}

func (s Set[T]) Diff(other Set[T]) Set[T] {
	result := NewSet[T]()
	for item := range s {
		if _, found := other[item]; !found {
			result.Add(item)
		}
	}
	return result
}
