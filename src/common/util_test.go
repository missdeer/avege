package common

import (
	"testing"
)

func TestIsFileExists(t *testing.T) {
	if b, e := IsFileExists("./util.go"); b == false || e != nil {
		t.Error("util.go should be here")
	}
	if b, e := IsFileExists("./dummy.go"); e != nil || b == true {
		t.Error("dummy.go shouldn't be here")
	}
}
