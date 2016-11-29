package k8s

import (
	"testing"
	"github.com/davecgh/go-spew/spew"
)

func TestList(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}

	spew.Dump(list)
}
