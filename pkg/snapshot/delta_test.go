package snapshot

import (
	"reflect"
	"testing"
)

func TestEliminates(t *testing.T) {
	d1 := Delta{
		prev: reflect.ValueOf("foo"),
		curr: reflect.ValueOf("bar"),
	}
	d2 := Delta{
		prev: reflect.ValueOf("bar"),
		curr: reflect.ValueOf("foo"),
	}
	if !d1.Eliminates(d2) {
		t.Errorf("expected d1 to eliminate d2")
	}
}
