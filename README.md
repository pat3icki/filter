# filter

[![Go Reference](https://pkg.go.dev/badge/github.com/pat3icki/filter.svg)](https://pkg.go.dev/github.com/pat3icki/filter)
[![Go Report Card](https://goreportcard.com/badge/github.com/pat3icki/filter)](https://goreportcard.com/report/github.com/pat3icki/filter)

`filter` is a high-performance Go library for copying and filtering fields between structs, slices, and pointers using struct tags. It supports tag-based field mapping, safe type conversions, deep/shallow pointer copying, and pre-compiled execution plans for maximum performance.

## Features

- **Tag-Based Mapping**: Match fields using configurable struct tags (defaults to `json`, supports `omitempty` or custom tags).
- **Field Filtering**: Ignore specific fields using `tag:"-"` or leave unexported fields untouched.
- **Type Conversion**: Automatically performs safe type conversions between compatible types (e.g. `int` to `int64`, `float32` to `float64`).
- **Flexible Copying**: Supports nested structs, slices (including slice element conversions), pointers, and nil pointer safety.
- **Deep or Shallow Copying**: Choose between default shallow copying or recursive deep copying via `WithDeepCopy()`.
- **Pre-Compiled Plans**: Build copy plans once with `BuildPlan` and re-execute them repeatedly for **14x faster performance** and minimal memory allocations.

---

## Installation

```bash
go get github.com/pat3icki/filter
```

---

## Quick Start

### Basic Struct Copy

```go
package main

import (
	"fmt"
	"github.com/pat3icki/filter"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"-"` // Ignored
}

type UserDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	src := User{ID: 1, Name: "Alice", Password: "secretpassword"}
	var dst UserDTO

	if err := filter.Copy(&dst, src); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", dst) // Output: {ID:1 Name:Alice}
}
```

### Selecting Required Fields into a Map (`Select`)

Use `Select` to dynamically extract specific fields from a struct (or pointer) or a map into a `map[string]any`, with optional exclusion (`ex ...string`):

```go
// Select required fields from struct minus excluded fields
srcStruct := User{ID: 1, Name: "Alice", Password: "secretpassword"}
fields, err := filter.Select([]string{"id", "name", "password"}, &srcStruct, "password")
// fields: map[id:1 name:Alice]

// Omit required (nil/empty) to select ALL fields except excluded ones
allMinusRole, err := filter.Select(nil, &srcStruct, "password")

// Select from map with excluded keys
srcMap := map[string]any{"id": 2, "name": "Bob", "email": "bob@example.com", "role": "admin"}
fieldsMap, err := filter.Select([]string{"name", "email", "role"}, srcMap, "role")
// fieldsMap: map[email:bob@example.com name:Bob]
```

---

## Configuration Options

### 1. Custom Struct Tags (`WithTagName`)

By default, `filter` uses the `json` tag. You can specify a custom tag:

```go
type Source struct {
	FullName string `db:"user_name"`
}

type Destination struct {
	Name string `db:"user_name"`
}

var dst Destination
err := filter.Copy(&dst, src, filter.WithTagName("db"))
```

### 2. Deep Copying (`WithDeepCopy`)

By default, pointer fields are shallow-copied. Use `WithDeepCopy()` to recursively duplicate nested structs, slices, and pointers:

```go
var dst Employee
err := filter.Copy(&dst, src, filter.WithDeepCopy())
```

---

## High Performance: Pre-Compiled Copy Plans

When executing repetitive copy operations (e.g., inside HTTP handlers or database loops), pre-compile a `Plan` using `BuildPlan` with generic destination and source types.

```go
// Pre-build plan once during initialization for UserDTO (Destination) and User (Source)
plan, err := filter.BuildPlan[UserDTO, User]()
if err != nil {
    log.Fatal(err)
}

// Re-execute plan on target instances for maximum performance
err = plan.Execute(&dst, src)
```

### Plan Metadata Introspection

Plans expose type description strings for source and destination types:

```go
fmt.Println(plan.Source)      // "main.User"
fmt.Println(plan.Destination) // "main.UserDTO"
```

---

## Benchmarks

Benchmark results comparing one-off `Copy` calls vs. pre-compiled `Plan.Execute()`:

| Operation | Execution Time (`ns/op`) | Memory (`B/op`) | Allocations (`allocs/op`) |
| :--- | :--- | :--- | :--- |
| **`Copy`** (Build + Execute) | ~17,584 ns | 2,856 B | 20 allocs |
| **`Plan.Execute`** (Pre-compiled) | **1,264 ns** | **48 B** | **2 allocs** |

---

## License

MIT License.
