package snapshot

import (
	"reflect"
	"testing"
)

func TestEliminates(t *testing.T) {
	type testCase struct {
		name     string
		d1       Delta
		d2       Delta
		expected bool
	}
	testCases := []testCase{
		{
			name: "same path, prev eliminates curr",
			d1: Delta{
				prev: reflect.ValueOf("foo"),
				curr: reflect.ValueOf("bar"),
			},
			d2: Delta{
				prev: reflect.ValueOf("bar"),
				curr: reflect.ValueOf("foo"),
			},
			expected: true,
		},
		{
			name: "different path, prev eliminates curr",
			d1: Delta{
				path: "abc",
				prev: reflect.ValueOf("foo"),
				curr: reflect.ValueOf("bar"),
			},
			d2: Delta{
				path: "abcdef",
				prev: reflect.ValueOf("bar"),
				curr: reflect.ValueOf("foo"),
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.d1.Eliminates(tc.d2) != tc.expected {
				t.Errorf("expected %v to eliminate %v", tc.d1, tc.d2)
			}
		})
	}
}
