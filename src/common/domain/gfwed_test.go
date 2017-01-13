package domain

import (
	"testing"
)

func TestIsGFWed(t *testing.T) {
	LoadDomainNameGFWed()

	if IsGFWed("www.twitter.com") == false {
		t.Error("twitter.com should be gfwed")
	}
}
