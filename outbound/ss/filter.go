package ss

import (
	"regexp"
)

// Filter is a set of regexp
// targeted to work around SSPanel policy
type Filter struct {
	pattern []*regexp.Regexp
}

// AddPattern to the pattern set
func (f *Filter) AddPattern(p *regexp.Regexp) {
	f.pattern = append(f.pattern, p)
}

// Match the byte slice
// Refer to regexp library
func (f *Filter) Match(b []byte) bool {
	for _, p := range f.pattern {
		if p.Match(b) {
			return true
		}
	}
	return false
}

// FindIndex of the left most match
// Refer to regexp library
func (f *Filter) FindIndex(b []byte) []int {
	var minloc []int
	for _, p := range f.pattern {
		loc := p.FindIndex(b)
		if loc != nil {
			if minloc == nil || loc[0] < minloc[0] {
				minloc = loc
			}
		}
	}
	return minloc
}

// Find the left most match
// Refer to regexp library
func (f *Filter) Find(b []byte) []byte {
	loc := f.FindIndex(b)
	if loc == nil {
		return nil
	}
	return b[loc[0]:loc[1]]
}
