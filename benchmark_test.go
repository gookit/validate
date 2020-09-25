package validate

import "testing"

func BenchmarkFieldSuccess(b *testing.B) {
	v := New(M{
		"name": "inhere",
	})
	v.StringRule("name", "len:6")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = v.Validate()
	}
}

func BenchmarkFieldSuccessParallel(b *testing.B) {
	v := New(M{
		"name": "inhere",
	})
	v.StringRule("name", "len:6")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = v.Validate()
		}
	})
}
