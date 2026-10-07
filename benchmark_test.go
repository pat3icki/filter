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

func BenchmarkPlan(b *testing.B) {
	type S struct {
		A int
		B string
		C float64
		D []int
		E *S
	}
	b.ResetTimer()
	for b.Loop() {
		_, _ = BuildPlan[S, S]()
	}
}

func BenchmarkSelect(b *testing.B) {
	type S struct {
		A int    `json:"a"`
		B string `json:"b"`
		C float64 `json:"c"`
	}
	srcStruct := S{A: 1, B: "test", C: 3.14}
	required := []string{"a", "b"}

	b.Run("Struct", func(b *testing.B) {
		for b.Loop() {
			_, _ = Select(required, srcStruct)
		}
	})

	srcMap := map[string]any{
		"a": 1,
		"b": "test",
		"c": 3.14,
	}

	b.Run("Map", func(b *testing.B) {
		for b.Loop() {
			_, _ = Select(required, srcMap)
		}
	})
}

