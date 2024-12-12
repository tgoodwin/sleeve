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

func (s Set[T]) Union(other Set[T]) Set[T] {
	result := NewSet[T]()
	for item := range s {
		result.Add(item)
	}
	for item := range other {
		result.Add(item)
	}
	return result
}

func (s Set[T]) List() []T {
	result := make([]T, 0, len(s))
	for item := range s {
		result = append(result, item)
	}
	return result
}
