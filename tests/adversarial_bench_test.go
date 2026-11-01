package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Each input has one field that needs coercion, which is how a caller selects the slow path.

type advItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type advWide struct {
	ID    int       `json:"id"`
	Items []advItem `json:"items"`
}

type advDeep struct {
	ID   int      `json:"id"`
	Next *advDeep `json:"next"`
}

func advWideItems(n int) []byte {
	var b strings.Builder
	b.WriteString(`{"id":"1","items":[{"name":"n","value":1}`)
	for i := 1; i < n; i++ {
		b.WriteString(`,{"name":"n","value":1}`)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

func advUnknownKeys(n int) []byte {
	var b strings.Builder
	b.WriteString(`{"id":"1"`)
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, `,"k%d":%d`, i, i)
	}
	b.WriteString(`}`)
	return []byte(b.String())
}

func advDeepNesting(levels int) []byte {
	return []byte(strings.Repeat(`{"id":"1","next":`, levels) + `null` + strings.Repeat(`}`, levels))
}

func BenchmarkAdversarial_WideItems(b *testing.B) {
	input := advWideItems(20000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := model.ParseInto[advWide](input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdversarial_UnknownKeys(b *testing.B) {
	input := advUnknownKeys(50000)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := model.ParseInto[advWide](input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdversarial_DeepNesting(b *testing.B) {
	input := advDeepNesting(30)
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := model.ParseInto[advDeep](input); err != nil {
			b.Fatal(err)
		}
	}
}
