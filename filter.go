package filter

import (
	"errors"
	"fmt"
	"reflect"
)

// Plan represents a pre‑built copy plan that can be executed later.
type Plan struct {
	Source      string
	Destination string
	dstType     reflect.Type
	srcType     reflect.Type
	dst         reflect.Value
	src         reflect.Value
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
	plan, err := BuildPlan(dst, src, opts...)
	if err != nil {
		return err
	}
	return plan.Execute()
}

// BuildPlan analyzes dst and src and builds a copy plan.
// It performs validation and creates a series of instructions.
// The plan can be executed multiple times (dst and src must remain valid).
// BuildPlan validates and builds the copy plan.
func BuildPlan(dst, src interface{}, opts ...Option) (*Plan, error) {
	cfg := &Config{TagName: "json"}
	for _, opt := range opts {
		opt(cfg)
	}

	dstV := reflect.ValueOf(dst)
	if dstV.Kind() != reflect.Ptr || dstV.IsNil() {
		return nil, fmt.Errorf("dst must be a non‑nil pointer, got %v", dstV.Kind())
	}
	dstVal := dstV.Elem()
	if !dstVal.CanSet() {
		return nil, errors.New("dst is not settable")
	}

	// dst base type must be struct or slice (after stripping pointers)
	dstBaseType := dstVal.Type()
	for dstBaseType.Kind() == reflect.Ptr {
		dstBaseType = dstBaseType.Elem()
	}
	dstBaseKind := dstBaseType.Kind()
	if dstBaseKind != reflect.Struct && dstBaseKind != reflect.Slice {
		return nil, fmt.Errorf("dst and src must be structs or slices, got %s", dstBaseKind)
	}

	srcV := reflect.ValueOf(src)
	// src base type must be struct or slice (after stripping pointers)
	srcBaseType := srcV.Type()
	for srcBaseType.Kind() == reflect.Ptr {
		srcBaseType = srcBaseType.Elem()
	}
	srcBaseKind := srcBaseType.Kind()
	if srcBaseKind != reflect.Struct && srcBaseKind != reflect.Slice {
		return nil, fmt.Errorf("dst and src must be structs or slices, got %s", srcBaseKind)
	}

	instr, err := buildTypeInstruction(dstVal.Type(), srcV.Type(), cfg)
	if err != nil {
		return nil, err
	}
	return &Plan{
		Source:      srcV.Type().String(),
		Destination: dstV.Type().String(),
		dstType:     dstVal.Type(),
		srcType:     srcV.Type(),
		dst:         dstVal,
		src:         srcV,
		instr:       instr,
		cfg:         cfg,
	}, nil
}

// Execute performs the copy according to the plan.
func (p *Plan) Execute() error {
	return executeTypeInstruction(p.instr, p.dst, p.src, p.cfg)
}
