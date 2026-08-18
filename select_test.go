package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type SelectUser struct {
	ID       int    `json:"id" db:"user_id"`
	Name     string `json:"name" db:"user_name"`
	Role     string `json:"role"`
	Secret   string `json:"-"`
	private  string
}

func TestSelect_BasicStructPointer(t *testing.T) {
	u := SelectUser{
		ID:     42,
		Name:   "Alice",
		Role:   "admin",
		Secret: "shh",
	}

	result, err := Select([]string{"id", "name"}, &u)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id":   42,
		"name": "Alice",
	}, result)
}

func TestSelect_WithExcept(t *testing.T) {
	u := SelectUser{
		ID:   42,
		Name: "Alice",
		Role: "admin",
	}

	// Select required fields minus except fields
	result, err := Select([]string{"id", "name", "role"}, &u, "role")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id":   42,
		"name": "Alice",
	}, result)
}

func TestSelect_NilRequiredWithExcept(t *testing.T) {
	u := SelectUser{
		ID:   42,
		Name: "Alice",
		Role: "admin",
	}

	// Nil/empty required selects all fields minus except
	result, err := Select(nil, &u, "role")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id":   42,
		"name": "Alice",
	}, result)
}

func TestSelect_MapWithExcept(t *testing.T) {
	m := map[string]any{
		"id":     100,
		"name":   "Bob",
		"email":  "bob@example.com",
		"active": true,
	}

	// Select required fields minus except
	result, err := Select([]string{"id", "name", "email"}, m, "email")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id":   100,
		"name": "Bob",
	}, result)

	// All fields minus except
	resultAll, err := Select(nil, m, "email", "active")
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id":   100,
		"name": "Bob",
	}, resultAll)
}

func TestSelect_StructValue(t *testing.T) {
	u := SelectUser{
		ID:   42,
		Name: "Alice",
	}

	result, err := Select([]string{"id", "name"}, u)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id":   42,
		"name": "Alice",
	}, result)
}

func TestSelect_NilMap(t *testing.T) {
	var m map[string]any = nil
	result, err := Select([]string{"id"}, m)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestSelect_FieldNameFallback(t *testing.T) {
	type Custom struct {
		Age int
	}
	c := Custom{Age: 30}

	result, err := Select([]string{"Age"}, &c)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"Age": 30,
	}, result)
}

func TestSelect_IgnoreAndUnexported(t *testing.T) {
	u := SelectUser{
		ID:      1,
		Secret:  "hidden",
		private: "unexported",
	}

	result, err := Select([]string{"Secret", "private", "id"}, &u)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"id": 1,
	}, result)
}

func TestSelect_NilPointerError(t *testing.T) {
	var u *SelectUser = nil
	_, err := Select([]string{"id"}, u)
	require.Error(t, err)
	require.EqualError(t, err, "src pointer is nil")
}

func TestSelect_UnsupportedTypeError(t *testing.T) {
	val := 123
	_, err := Select([]string{"id"}, val)
	require.Error(t, err)
	require.Contains(t, err.Error(), "src must be a struct or map")
}
