package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- Test data structures ----------

type Person struct {
	Name   string `json:"name"`
	Age    int    `json:"age"`
	hidden string // unexported, should be ignored
}

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type Employee struct {
	Person   Person   `json:"person"`
	Address  Address  `json:"address"`
	Salary   float64  `json:"salary"`
	Tags     []string `json:"tags"`
	Manager  *Person  `json:"manager"`
	ID       int      `json:"id"`
	Ignored  string   `json:"-"`
	unexport string
}

// ---------- Basic copy ----------

func TestCopy_BasicStruct(t *testing.T) {
	src := Employee{
		Person:  Person{Name: "Alice", Age: 30},
		Address: Address{Street: "123 Main", City: "Springfield"},
		Salary:  50000.0,
		Tags:    []string{"go", "test"},
		Manager: &Person{Name: "Bob", Age: 40},
		ID:      42,
		Ignored: "ignored",
	}
	var dst Employee
	err := Copy(&dst, src)
	require.NoError(t, err)

	// Manager pointer should be shallow copied (default), so they point to same object
	require.Same(t, src.Manager, dst.Manager)
	// Ignored field should be empty string (zero value)
	require.Empty(t, dst.Ignored)
	// Unexported field should be zero
	require.Empty(t, dst.unexport)
	// Compare other fields (deep equal for nested structs)
	require.Equal(t, src.Person, dst.Person)
	require.Equal(t, src.Address, dst.Address)
	require.Equal(t, src.Salary, dst.Salary)
	require.Equal(t, src.Tags, dst.Tags)
	require.Equal(t, src.ID, dst.ID)
}

// ---------- Copy with tags and custom tag name ----------

func TestCopy_WithTagName(t *testing.T) {
	type Src struct {
		FullName string `json:"full_name"`
		Years    int    `json:"years"`
	}
	type Dst struct {
		Name string `json:"full_name"`
		Age  int    `json:"years"`
	}
	src := Src{FullName: "John Doe", Years: 25}
	var dst Dst
	err := Copy(&dst, src, WithTagName("json"))
	require.NoError(t, err)
	require.Equal(t, "John Doe", dst.Name)
	require.Equal(t, 25, dst.Age)
}

// ---------- Ignored fields ----------

func TestCopy_IgnoreTag(t *testing.T) {
	type Src struct {
		Value  int    `json:"value"`
		Secret string `json:"-"`
	}
	type Dst struct {
		Value  int    `json:"value"`
		Secret string `json:"-"`
	}
	src := Src{Value: 123, Secret: "hidden"}
	var dst Dst
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Equal(t, 123, dst.Value)
	require.Empty(t, dst.Secret)
}

// ---------- Unexported fields ----------

func TestCopy_UnexportedFields(t *testing.T) {
	type Src struct {
		Public  int
		private string
	}
	type Dst struct {
		Public  int
		private string
	}
	src := Src{Public: 42, private: "should not copy"}
	var dst Dst
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Equal(t, 42, dst.Public)
	require.Empty(t, dst.private)
}

// ---------- Type conversion ----------

func TestCopy_TypeConversion(t *testing.T) {
	type Src struct {
		IntVal  int     `json:"int"`
		Float32 float32 `json:"float"`
	}
	type Dst struct {
		IntVal  int64   `json:"int"`
		Float32 float64 `json:"float"`
	}
	src := Src{IntVal: 100, Float32: 0.5}
	var dst Dst
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Equal(t, int64(100), dst.IntVal)
	require.Equal(t, float64(0.5), dst.Float32) // exact representation
}

// ---------- Slice copy (shallow) ----------

func TestCopy_SliceShallow(t *testing.T) {
	src := []int{1, 2, 3}
	var dst []int
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Equal(t, src, dst)

	// Modify dst and ensure src unchanged (since it's a new slice)
	dst[0] = 99
	require.NotEqual(t, src[0], 99, "Src was modified, shallow copy should have created new slice")
}

// ---------- Slice copy with structs (deep copy of elements) ----------

func TestCopy_SliceOfStructs_DeepElements(t *testing.T) {
	type Item struct {
		ID   int
		Name string
	}
	src := []Item{{1, "one"}, {2, "two"}}
	var dst []Item
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Equal(t, src, dst)

	// Modify dst element and ensure src unchanged
	dst[0].Name = "changed"
	require.NotEqual(t, src[0].Name, "changed")
}

// ---------- Pointer copy (shallow) ----------

func TestCopy_PointerShallow(t *testing.T) {
	src := &Person{Name: "Alice", Age: 30}
	var dst *Person
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Same(t, src, dst)

	// Modify through dst, src should reflect.
	dst.Age = 99
	require.Equal(t, 99, src.Age)
}

// ---------- Pointer copy (deep) ----------

func TestCopy_PointerDeep(t *testing.T) {
	src := &Person{Name: "Alice", Age: 30}
	var dst *Person
	err := Copy(&dst, src, WithDeepCopy())
	require.NoError(t, err)
	require.NotSame(t, src, dst)
	require.Equal(t, src.Name, dst.Name)
	require.Equal(t, src.Age, dst.Age)

	// Modify dst, src unchanged
	dst.Age = 99
	require.NotEqual(t, src.Age, 99)
}

// ---------- Nested structures with deep copy ----------

func TestCopy_NestedStructDeep(t *testing.T) {
	type Inner struct {
		Val int
	}
	type Outer struct {
		Inner *Inner
	}
	src := Outer{Inner: &Inner{Val: 42}}
	var dst Outer
	err := Copy(&dst, src, WithDeepCopy())
	require.NoError(t, err)
	require.NotSame(t, src.Inner, dst.Inner)
	require.Equal(t, 42, dst.Inner.Val)

	dst.Inner.Val = 99
	require.NotEqual(t, src.Inner.Val, 99)
}

// ---------- Nested slices with deep copy ----------

func TestCopy_NestedSliceDeep(t *testing.T) {
	type Outer struct {
		Slice []int
	}
	src := Outer{Slice: []int{1, 2, 3}}
	var dst Outer
	err := Copy(&dst, src, WithDeepCopy())
	require.NoError(t, err)
	// Ensure new slice created
	require.NotSame(t, &src.Slice[0], &dst.Slice[0])

	dst.Slice[0] = 99
	require.NotEqual(t, src.Slice[0], 99)
}

// ---------- Nil source pointer ----------

func TestCopy_NilSourcePointer(t *testing.T) {
	type Data struct {
		X int
	}
	var src *Data = nil
	var dst Data
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Zero(t, dst.X)

	// Also test when dst is a pointer to struct
	var dstPtr *Data
	err = Copy(&dstPtr, src)
	require.NoError(t, err)
	require.Nil(t, dstPtr)
}

// ---------- Nil source pointer to slice ----------

func TestCopy_NilSourceSlicePointer(t *testing.T) {
	var src *[]int = nil
	var dst []int
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Nil(t, dst)
}

// ---------- Error cases ----------

func TestCopy_Errors(t *testing.T) {
	// dst not a pointer
	var dst Employee
	err := Copy(dst, Employee{})
	require.Error(t, err)
	require.EqualError(t, err, "dst must be a non‑nil pointer, got struct")

	// dst nil pointer
	err = Copy(nil, Employee{})
	require.Error(t, err)
	require.EqualError(t, err, "dst must be a non‑nil pointer, got invalid")

	// Kind mismatch: dst struct, src slice
	var dstStruct Employee
	err = Copy(&dstStruct, []int{1, 2, 3})
	require.Error(t, err)
	require.EqualError(t, err, "dst (struct) and src (slice) kinds must match")

	// dst not struct or slice (e.g., int)
	var dstInt int
	err = Copy(&dstInt, 42)
	require.Error(t, err)
	require.EqualError(t, err, "dst and src must be structs or slices, got int")

	// Incompatible types in struct field (e.g., string vs int) - should error during buildInstruction
	type SrcBad struct {
		F int
	}
	type DstBad struct {
		F string
	}
	srcBad := SrcBad{F: 123}
	var dstBad DstBad
	err = Copy(&dstBad, srcBad)
	require.Error(t, err)
}

// ---------- Plan reuse ----------

func TestPlan_Execute(t *testing.T) {
	type S struct {
		A int
		B string
	}
	src := S{A: 10, B: "hello"}
	var dst S
	plan, err := BuildPlan(&dst, src)
	require.NoError(t, err)

	// Execute
	err = plan.Execute()
	require.NoError(t, err)
	require.Equal(t, S{A: 10, B: "hello"}, dst)

	// Change src and execute again (plan uses same dst, src values are evaluated at plan build time)
	src.A = 99
	err = plan.Execute()
	require.NoError(t, err)
	// dst should remain 10 because src was captured by value
	require.Equal(t, 10, dst.A)
}

// ---------- Options ----------

func TestOptions(t *testing.T) {
	// Test WithDeepCopy as already done, but also test combined options
	type PtrStruct struct {
		P *int
	}
	src := PtrStruct{P: new(int)}
	*src.P = 42
	var dst PtrStruct
	err := Copy(&dst, src, WithDeepCopy(), WithTagName("json"))
	require.NoError(t, err)
	require.NotSame(t, src.P, dst.P)
}

// ---------- Edge cases: empty struct, empty slice ----------

func TestCopy_EmptyStruct(t *testing.T) {
	type Empty struct{}
	src := Empty{}
	var dst Empty
	err := Copy(&dst, src)
	require.NoError(t, err)
	// no fields to compare, just ensure no error
}

func TestCopy_EmptySlice(t *testing.T) {
	src := []int{}
	var dst []int
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Empty(t, dst)
}

// ---------- Multiple levels of nesting ----------

func TestCopy_DeepNesting(t *testing.T) {
	type A struct {
		X int
	}
	type B struct {
		A *A
	}
	type C struct {
		B B
	}
	src := C{B: B{A: &A{X: 5}}}
	var dst C
	err := Copy(&dst, src, WithDeepCopy())
	require.NoError(t, err)
	require.NotSame(t, src.B.A, dst.B.A)
	require.Equal(t, 5, dst.B.A.X)

	dst.B.A.X = 99
	require.NotEqual(t, src.B.A.X, 99)
}

// ---------- Copy with source as pointer to pointer ----------

func TestCopy_PointerToPointer(t *testing.T) {
	type S struct {
		V int
	}
	srcPtr := &S{V: 7}
	srcPtrPtr := &srcPtr
	var dst *S
	// dst is a pointer to S, src is **S. The code will dereference pointers until non-pointer or nil.
	err := Copy(&dst, srcPtrPtr)
	require.NoError(t, err)
	require.Same(t, srcPtr, dst)
	require.Equal(t, 7, dst.V)
}

// ---------- Plan introspection metadata ----------

func TestPlan_Metadata(t *testing.T) {
	type Src struct {
		Val int `json:"val"`
	}
	type Dst struct {
		Val int `json:"val"`
	}
	src := Src{Val: 10}
	var dst Dst
	plan, err := BuildPlan(&dst, src)
	require.NoError(t, err)
	require.Equal(t, "filter.Src", plan.Source)
	require.Equal(t, "*filter.Dst", plan.Destination)
}

// ---------- Slice conversion ----------

func TestCopy_SliceConversion(t *testing.T) {
	src := []int{10, 20, 30}
	var dst []int64
	err := Copy(&dst, src)
	require.NoError(t, err)
	require.Equal(t, []int64{10, 20, 30}, dst)
}

// ---------- Benchmark ----------

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
		_ = Copy(&dst, src)
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
	plan, err := BuildPlan(&dst, src)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = plan.Execute()
	}
}
