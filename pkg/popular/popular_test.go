package popular

import (
	"fmt"
	"testing"
	"time"
)

var (
	tests = []struct {
		word string
	}{
		{word: "aaa"},
		{word: "bbb"},
		{word: "ccc"},
		{word: "aaa"},
		{word: "bbb"},
		{word: "ccc"},
		{word: "aaa"},
		{word: "bbb"},
		{word: "ccc"},
		{word: "ccc"},
	}
)

// 获取1秒内高频词
func TestMostPopularWord_GetTopWord_1000ms(t *testing.T) {
	most := NewMostPopularWord(time.Second)
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			most.Record(tt.word)
			time.Sleep(time.Millisecond)
		})
	}

	want := "ccc"
	cutDura := time.Second
	if got := most.GetTopWord(time.Second); got != want {
		t.Errorf(`GetTopWord(%s) want "%s", but got "%s"`, cutDura, want, got)
	}
}

// 获取500毫秒内高频词
func TestMostPopularWord_GetTopWord_500ms(t *testing.T) {
	most := NewMostPopularWord(time.Second)
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			most.Record(tt.word)
			time.Sleep(100 * time.Millisecond)
		})
	}

	want := "ccc"
	cutDura := 500 * time.Millisecond
	if got := most.GetTopWord(cutDura); got != want {
		t.Errorf(`GetTopWord(%s) want "%s", but got "%s"`, cutDura, want, got)
	}
}

func BenchmarkMostPopularWord_Record(b *testing.B) {
	for i := 0; i < b.N; i++ {
		most := NewMostPopularWord(time.Second)
		for j := 0; j < 1000; j++ {
			for _, tt := range tests {
				most.Record(tt.word)
			}
		}
	}
}

var benchWant string

func BenchmarkMostPopularWord_GetTopWord1(b *testing.B) {
	cutDura := 1 * time.Second
	most := NewMostPopularWord(cutDura)
	for j := 0; j < 10000; j++ {
		for _, tt := range tests {
			most.Record(tt.word)
		}
	}

	for i := 0; i < b.N; i++ {
		benchWant = most.GetTopWord(cutDura)
	}
	fmt.Println("BenchmarkMostPopularWord_GetTopWord1: ", benchWant)
}

func BenchmarkMostPopularWord_GetTopWord2(b *testing.B) {
	cutDura := 1 * time.Second
	most := NewMostPopularWord(cutDura)
	for j := 0; j < 10000; j++ {
		for _, tt := range tests {
			most.Record(tt.word)
		}
	}

	// 测试过期情况
	for i := 0; i < 100000000; i++ {
		benchWant = most.GetTopWord(cutDura)
	}
	fmt.Println("BenchmarkMostPopularWord_GetTopWord2: ", benchWant)
}
