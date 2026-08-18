package filter

import "testing"

func BenchmarkCopy(b *testing.B) {
	type S struct {
		A int
		B string
		C float64
		D []int
		E *S
	}
	src := S{A: 1, B: "test", C: 3.14, D: []int{1, 2, 3}, E: &S{A: 2}}
	var dst S
	b.ResetTimer()
	for b.Loop() {
		_ = Copy(&dst, &src)
	}
}

func BenchmarkPlanExecute(b *testing.B) {
	type S struct {
		A int
		B string
		C float64
		D []int
		E *S
	}
	src := S{A: 1, B: "test", C: 3.14, D: []int{1, 2, 3}, E: &S{A: 2}}
	var dst S
	plan, err := BuildPlan[S, S]()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = plan.Execute(&dst, &src)
	}
}
