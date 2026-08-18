package filter

import (
	"errors"
	"fmt"
	"reflect"
)

// Plan represents a pre‑built copy plan that can be executed later.
type Plan[D any, S any] struct {
	Source      string
	Destination string
	dstType     reflect.Type
	srcType     reflect.Type
	instr       instruction // root instruction
	cfg         *Config     // stored configuration
}

// instruction describes a copy operation.
type instruction struct {
	kind     reflect.Kind
	direct   bool
	dstType  reflect.Type
	srcType  reflect.Type
	dstIndex []int
	srcIndex []int

	fields    []instruction
	elemInstr *instruction
	ptrInstr  *instruction
}

// Config holds options for the copy operation.
type Config struct {
	DeepCopy bool   // whether to perform deep copying of nested structures
	TagName  string // tag name to use for field mapping (default: "json")
}

// Option is a functional option for configuring Copy.
type Option func(*Config)

// WithDeepCopy enables deep copying of nested structs, slices, and pointers.
func WithDeepCopy() Option {
	return func(c *Config) {
		c.DeepCopy = true
	}
}

// WithTagName sets the tag name used to identify fields (default "json").
func WithTagName(tag string) Option {
	return func(c *Config) {
		c.TagName = tag
	}
}

// Copy copies fields from src to dst using the specified tag.
// Rules:
//   - dst must be a non‑nil pointer to a struct or slice.
//   - src may be a value or pointer to a struct or slice.
//   - Fields are matched by their tag name (or field name if no tag).
//   - Fields tagged with `tag:"-"` are ignored.
//   - Unexported fields are ignored.
//   - Supports nested structs, slices, and pointer indirection.
//   - Performs safe type conversion when types are convertible.
//   - If src is a nil pointer, dst is zeroed (shallow or deep depending on options).
//   - WithDeepCopy enabled, pointers are dereferenced recursively and copied;
//     otherwise they are assigned directly (shallow).
func Copy(dst, src interface{}, opts ...Option) error {
	cfg := &Config{TagName: "json"}
	for _, opt := range opts {
		opt(cfg)
	}

	dstV := reflect.ValueOf(dst)
	if dstV.Kind() != reflect.Ptr || dstV.IsNil() {
		return fmt.Errorf("dst must be a non‑nil pointer, got %v", dstV.Kind())
	}
	dstVal := dstV.Elem()
	if !dstVal.CanSet() {
		return errors.New("dst is not settable")
	}

	dstBaseType := dstVal.Type()
	for dstBaseType.Kind() == reflect.Ptr {
		dstBaseType = dstBaseType.Elem()
	}
	dstBaseKind := dstBaseType.Kind()
	if dstBaseKind != reflect.Struct && dstBaseKind != reflect.Slice {
		return fmt.Errorf("dst and src must be structs or slices, got %s", dstBaseKind)
	}

	srcV := reflect.ValueOf(src)
	srcBaseType := srcV.Type()
	for srcBaseType.Kind() == reflect.Ptr {
		srcBaseType = srcBaseType.Elem()
	}
	srcBaseKind := srcBaseType.Kind()
	if srcBaseKind != reflect.Struct && srcBaseKind != reflect.Slice {
		return fmt.Errorf("dst and src must be structs or slices, got %s", srcBaseKind)
	}

	instr, err := buildTypeInstruction(dstVal.Type(), srcV.Type(), cfg)
	if err != nil {
		return err
	}
	return executeTypeInstruction(instr, dstVal, srcV, cfg)
}

// BuildPlan analyzes types D and S and builds a copy plan.
// It performs validation and creates a series of instructions.
// The plan can be executed multiple times with different dst and src values of type D and S.
func BuildPlan[D any, S any](opts ...Option) (*Plan[D, S], error) {
	cfg := &Config{TagName: "json"}
	for _, opt := range opts {
		opt(cfg)
	}

	dstType := reflect.TypeFor[D]()
	srcType := reflect.TypeFor[S]()

	dstBaseType := dstType
	for dstBaseType.Kind() == reflect.Ptr {
		dstBaseType = dstBaseType.Elem()
	}
	dstBaseKind := dstBaseType.Kind()
	if dstBaseKind != reflect.Struct && dstBaseKind != reflect.Slice {
		return nil, fmt.Errorf("dst and src must be structs or slices, got %s", dstBaseKind)
	}

	srcBaseType := srcType
	for srcBaseType.Kind() == reflect.Ptr {
		srcBaseType = srcBaseType.Elem()
	}
	srcBaseKind := srcBaseType.Kind()
	if srcBaseKind != reflect.Struct && srcBaseKind != reflect.Slice {
		return nil, fmt.Errorf("dst and src must be structs or slices, got %s", srcBaseKind)
	}

	instr, err := buildTypeInstruction(dstType, srcType, cfg)
	if err != nil {
		return nil, err
	}
	return &Plan[D, S]{
		Source:      srcType.String(),
		Destination: dstType.String(),
		dstType:     dstType,
		srcType:     srcType,
		instr:       instr,
		cfg:         cfg,
	}, nil
}

// Execute performs the copy according to the plan for the given dst pointer and src value.
func (p *Plan[D, S]) Execute(dst *D, src *S) error {
	if dst == nil {
		return errors.New("dst must be a non‑nil pointer")
	}
	dstV := reflect.ValueOf(dst).Elem()
	if !dstV.CanSet() {
		return errors.New("dst is not settable")
	}
	srcV := reflect.ValueOf(src)
	return executeTypeInstruction(p.instr, dstV, srcV, p.cfg)
}
