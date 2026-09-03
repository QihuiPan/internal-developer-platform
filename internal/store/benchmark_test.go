package store

import (
	"fmt"
	"path/filepath"
	"testing"
)

func BenchmarkCreateServiceDurableTransaction(b *testing.B) {
	root := b.TempDir()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		state, err := Open(filepath.Join(root, fmt.Sprintf("state-%d.json", index)))
		if err != nil {
			b.Fatal(err)
		}
		if _, err := state.CreateService("benchmark", "benchmark", fmt.Sprintf("request-%08d", index), descriptor("benchmark-service")); err != nil {
			b.Fatal(err)
		}
	}
}
