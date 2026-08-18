// Package filter provides a high-performance Go library for copying and filtering fields
// between structs, slices, pointers, and maps using struct tags.
//
// Key Features:
//
//   - Copy: Performs one-off copying between structs, slices, or pointers with tag-based matching
//     (defaulting to "json"), automatic type conversion, and configurable pointer dereferencing.
//
//   - Pre-Compiled Plans (BuildPlan & Plan[D, S]): Pre-compiles field instructions based on destination
//     type D and source type S. Plans can be re-executed repeatedly via Plan.Execute(&dst, src) for maximum
//     performance and minimal memory allocations.
//
//   - Select: Extracts required fields from structs or maps into a map[string]any, supporting field
//     exclusion via variadic arguments.
//
//   - Options: Customize copying behavior using WithTagName(tag) for custom struct tags or WithDeepCopy()
//     for recursive pointer and slice deep copying.
//
// Basic Copy Example:
//
//	type User struct {
//		ID       int    `json:"id"`
//		Name     string `json:"name"`
//		Password string `json:"-"`
//	}
//
//	type UserDTO struct {
//		ID   int    `json:"id"`
//		Name string `json:"name"`
//	}
//
//	var dst UserDTO
//	src := User{ID: 1, Name: "Alice", Password: "secret"}
//	err := filter.Copy(&dst, src)
//
// Generic Pre-Compiled Plan Example:
//
//	plan, err := filter.BuildPlan[UserDTO, User]()
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = plan.Execute(&dst, src)
//
// Select Example:
//
//	fields, err := filter.Select([]string{"id", "name"}, &src, "password")
package filter
