package guard

import (
	"strings"
	"testing"
)

// BenchmarkNormalizeAll24KB and BenchmarkNormalizeAll8MiB print MB/s via
// b.SetBytes, making the two required input sizes reproducible with:
// go test -run '^$' -bench 'BenchmarkNormalizeAll(24KB|8MiB)$' -benchtime=5x
func BenchmarkNormalizeAll24KB(b *testing.B) {
	benchmarkNormalizeAll(b, 24<<10)
}

func BenchmarkNormalizeAll8MiB(b *testing.B) {
	benchmarkNormalizeAll(b, 8<<20)
}

func benchmarkNormalizeAll(b *testing.B, size int) {
	input := strings.Repeat("A", size)
	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NormalizeAll(input, nil)
	}
}
