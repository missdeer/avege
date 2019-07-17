package ip

import (
	"testing"
)

func TestInChina(t *testing.T) {
	LoadChinaIPList(false)

	if InChina("208.67.222.222") == true {
		t.Error("208.67.222.222 should be foreign IP")
	}

	if InChina("8.8.8.8") == true {
		t.Error("8.8.8.8 should be foreign IP")
	}

	if InChina("114.114.114.114") == false {
		t.Error("114.114.114.114 should be in China")
	}

	if InChina("223.5.5.5") == false {
		t.Error("223.5.5.5 should be in China")
	}
}

func BenchmarkOutOfChina(b *testing.B) {
	LoadChinaIPList(false)
	for i := 0; i < b.N; i++ {
		InChina("208.67.222.222")
	}
}

func BenchmarkInChina(b *testing.B) {
	LoadChinaIPList(false)
	for i := 0; i < b.N; i++ {
		InChina("114.114.114.114")
	}
}
