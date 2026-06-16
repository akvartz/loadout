package diff

import (
	"fmt"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

func BenchmarkUnionKeys(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			m1 := make(map[string]state.SourceState, n)
			m2 := make(map[string]state.SourceState, n)
			for i := 0; i < n; i++ {
				m1[fmt.Sprintf("key-%d", i)] = state.SourceState{}
				// m2 has 50% overlap
				m2[fmt.Sprintf("key-%d", i+n/2)] = state.SourceState{}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = unionKeys(m1, m2)
			}
		})
	}
}
