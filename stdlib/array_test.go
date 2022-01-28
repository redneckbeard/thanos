package stdlib

import (
	"reflect"
	"testing"
)

func TestSubtractSlice(t *testing.T) {
	strLeft := []string{"a", "b", "e", "c", "d", "e"}
	strRight := []string{"b", "e"}
	strResult := SubtractSlice(strLeft, strRight)
	if !reflect.DeepEqual(strResult, []string{"a", "c", "d"}) {
		t.Fatal("SubtractSlice failed for []string args")
	}
	intLeft := []int{1, 2, 2, 4, 5, 6, 2, 8}
	intRight := []int{2, 6}
	intResult := SubtractSlice(intLeft, intRight)
	if !reflect.DeepEqual(intResult, []int{1, 4, 5, 8}) {
		t.Fatal("SubtractSlice failed for []int args")
	}
}
