package filter

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// buildTypeInstruction builds static instructions based on reflect.Type without binding values.
func buildTypeInstruction(dstType, srcType reflect.Type, cfg *Config) (instruction, error) {
	if dstType.Kind() == reflect.Ptr {
		return buildPointerTypeInstruction(dstType, srcType, cfg)
	}

	for srcType.Kind() == reflect.Ptr {
		srcType = srcType.Elem()
	}

	if dstType.Kind() != srcType.Kind() {
		if srcType.ConvertibleTo(dstType) {
			isIntToString := dstType.Kind() == reflect.String && srcType.Kind() >= reflect.Int && srcType.Kind() <= reflect.Uintptr
			if !isIntToString {
				return instruction{
					kind:    dstType.Kind(),
					direct:  true,
					dstType: dstType,
					srcType: srcType,
				}, nil
			}
		}
		return instruction{}, fmt.Errorf("dst (%s) and src (%s) kinds must match", dstType.Kind(), srcType.Kind())
	}

	switch dstType.Kind() {
	case reflect.Struct:
		return buildStructTypeInstruction(dstType, srcType, cfg)
	case reflect.Slice:
		return buildSliceTypeInstruction(dstType, srcType, cfg)
	default:
		if srcType.ConvertibleTo(dstType) {
			return instruction{
				kind:    dstType.Kind(),
				direct:  true,
				dstType: dstType,
				srcType: srcType,
			}, nil
		}
		return instruction{}, fmt.Errorf("cannot assign %s to %s", srcType, dstType)
	}
}

// buildPointerTypeInstruction pre-compiles instructions for pointer indirection.
func buildPointerTypeInstruction(dstType, srcType reflect.Type, cfg *Config) (instruction, error) {
	if !cfg.DeepCopy {
		srcValType := srcType
		for srcValType.Kind() == reflect.Ptr && !srcValType.AssignableTo(dstType) {
			srcValType = srcValType.Elem()
		}
		if srcValType.AssignableTo(dstType) {
			return instruction{
				kind:    reflect.Ptr,
				direct:  true,
				dstType: dstType,
				srcType: srcValType,
			}, nil
		}
		if srcValType.AssignableTo(dstType.Elem()) {
			return instruction{
				kind:    reflect.Ptr,
				direct:  true,
				dstType: dstType.Elem(),
				srcType: srcValType,
			}, nil
		}
		return instruction{}, fmt.Errorf("cannot assign %s to %s", srcType, dstType)
	}

	// Deep copy: allocate new element and copy recursively.
	elemType := dstType.Elem()
	var srcElemType reflect.Type
	if srcType.Kind() == reflect.Ptr {
		srcElemType = srcType.Elem()
	} else {
		srcElemType = srcType
	}
	sub, err := buildTypeInstruction(elemType, srcElemType, cfg)
	if err != nil {
		return instruction{}, err
	}
	return instruction{
		kind:     reflect.Ptr,
		direct:   false,
		dstType:  dstType,
		srcType:  srcType,
		ptrInstr: &sub,
	}, nil
}

// buildStructTypeInstruction builds instruction tree for struct fields based on types.
func buildStructTypeInstruction(dstType, srcType reflect.Type, cfg *Config) (instruction, error) {
	if dstType.Kind() != reflect.Struct || srcType.Kind() != reflect.Struct {
		return instruction{}, errors.New("not structs")
	}

	srcFieldMap := make(map[string]reflect.StructField)
	for i := 0; i < srcType.NumField(); i++ {
		f := srcType.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		tag := f.Tag.Get(cfg.TagName)
		if tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = f.Name
		}
		srcFieldMap[name] = f
	}

	var fieldInstrs []instruction

	for i := 0; i < dstType.NumField(); i++ {
		dstField := dstType.Field(i)
		if dstField.PkgPath != "" { // unexported
			continue
		}
		tag := dstField.Tag.Get(cfg.TagName)
		if tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = dstField.Name
		}

		srcField, ok := srcFieldMap[name]
		if !ok {
			continue
		}

		sub, err := buildTypeInstruction(dstField.Type, srcField.Type, cfg)
		if err != nil {
			return instruction{}, fmt.Errorf("field %s: %w", name, err)
		}
		sub.dstIndex = dstField.Index
		sub.srcIndex = srcField.Index
		fieldInstrs = append(fieldInstrs, sub)
	}

	return instruction{
		kind:    reflect.Struct,
		direct:  false,
		dstType: dstType,
		srcType: srcType,
		fields:  fieldInstrs,
	}, nil
}

// buildSliceTypeInstruction pre-compiles element instruction for slice copying.
func buildSliceTypeInstruction(dstType, srcType reflect.Type, cfg *Config) (instruction, error) {
	if dstType.Kind() != reflect.Slice || srcType.Kind() != reflect.Slice {
		return instruction{}, errors.New("not slices")
	}

	dstElemType := dstType.Elem()
	srcElemType := srcType.Elem()

	elemSub, err := buildTypeInstruction(dstElemType, srcElemType, cfg)
	if err != nil {
		return instruction{}, fmt.Errorf("slice element error: %w", err)
	}

	return instruction{
		kind:      reflect.Slice,
		direct:    false,
		dstType:   dstType,
		srcType:   srcType,
		elemInstr: &elemSub,
	}, nil
}

// executeTypeInstruction executes the copy operation using pre-built instructions and current target values.
func executeTypeInstruction(instr instruction, dst, src reflect.Value, cfg *Config) error {
	for src.Kind() == reflect.Ptr && instr.kind != reflect.Ptr {
		if src.IsNil() {
			if dst.CanSet() {
				dst.Set(reflect.Zero(dst.Type()))
			}
			return nil
		}
		src = src.Elem()
	}

	if instr.direct {
		if !dst.CanSet() {
			return errors.New("cannot set destination")
		}
		srcVal := src
		for srcVal.Kind() == reflect.Ptr && !srcVal.Type().AssignableTo(dst.Type()) && !srcVal.Type().ConvertibleTo(dst.Type()) {
			if srcVal.IsNil() {
				srcVal = reflect.Zero(dst.Type())
				break
			}
			srcVal = srcVal.Elem()
		}
		if !srcVal.Type().AssignableTo(dst.Type()) {
			if srcVal.Type().ConvertibleTo(dst.Type()) {
				srcVal = srcVal.Convert(dst.Type())
			} else {
				return fmt.Errorf("cannot assign %s to %s", srcVal.Type(), dst.Type())
			}
		}
		dst.Set(srcVal)
		return nil
	}

	switch instr.kind {
	case reflect.Ptr:
		if src.IsNil() {
			if dst.CanSet() {
				dst.Set(reflect.Zero(dst.Type()))
			}
			return nil
		}
		if !cfg.DeepCopy {
			srcVal := src
			for srcVal.Kind() == reflect.Ptr && !srcVal.Type().AssignableTo(dst.Type()) {
				srcVal = srcVal.Elem()
			}
			if srcVal.Type().AssignableTo(dst.Type()) {
				dst.Set(srcVal)
				return nil
			}
			if srcVal.Type().AssignableTo(dst.Type().Elem()) {
				dst.Elem().Set(srcVal)
				return nil
			}
			return fmt.Errorf("cannot assign %s to %s", src.Type(), dst.Type())
		}

		elemType := dst.Type().Elem()
		newDst := reflect.New(elemType).Elem()
		srcElem := src.Elem()
		if err := executeTypeInstruction(*instr.ptrInstr, newDst, srcElem, cfg); err != nil {
			return err
		}
		ptrVal := newDst.Addr()
		dst.Set(ptrVal)
		return nil

	case reflect.Struct:
		for _, fi := range instr.fields {
			dstFieldVal := dst.FieldByIndex(fi.dstIndex)
			srcFieldVal := src.FieldByIndex(fi.srcIndex)
			if err := executeTypeInstruction(fi, dstFieldVal, srcFieldVal, cfg); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice:
		srcLen := src.Len()
		newSlice := reflect.MakeSlice(dst.Type(), srcLen, srcLen)
		for i := 0; i < srcLen; i++ {
			srcElem := src.Index(i)
			dstElem := newSlice.Index(i)
			if err := executeTypeInstruction(*instr.elemInstr, dstElem, srcElem, cfg); err != nil {
				return err
			}
		}
		if !dst.CanSet() {
			return errors.New("cannot set destination slice")
		}
		dst.Set(newSlice)
		return nil

	default:
		return fmt.Errorf("unsupported kind in instruction: %s", instr.kind)
	}
}
