package filter

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Select returns a map containing fields from src.
// Required fields (if provided) are included, minus any fields listed in ex.
// If required is empty or nil, all available fields from src are included except those listed in ex.
// src can be a struct, pointer to a struct, map, or pointer to a map.
func Select[T any](required []string, src T, ex ...string) (map[string]any, error) {
	var srcAny any = src
	if srcAny == nil {
		return nil, errors.New("src cannot be nil")
	}

	val := reflect.ValueOf(src)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, errors.New("src pointer is nil")
		}
		val = val.Elem()
	}

	var exceptMap map[string]bool
	if len(ex) > 0 {
		exceptMap = make(map[string]bool, len(ex))
		for _, e := range ex {
			exceptMap[e] = true
		}
	}

	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		numField := typ.NumField()

		capacity := numField
		if len(required) > 0 {
			capacity = len(required)
		}
		result := make(map[string]any, capacity)

		if len(required) == 0 {
			for i := 0; i < numField; i++ {
				field := typ.Field(i)
				if field.PkgPath != "" { // Skip unexported fields
					continue
				}

				tag := field.Tag.Get("json")
				if tag == "-" {
					continue
				}

				tagName := tag
				if idx := strings.IndexByte(tag, ','); idx != -1 {
					tagName = tag[:idx]
				}
				key := tagName
				if key == "" {
					key = field.Name
				}

				if exceptMap != nil && exceptMap[key] {
					continue
				}
				result[key] = val.Field(i).Interface()
			}
		} else {
			for _, req := range required {
				if exceptMap != nil && exceptMap[req] {
					continue
				}
				found := false
				for i := 0; i < numField; i++ {
					field := typ.Field(i)
					if field.PkgPath != "" {
						continue
					}
					tag := field.Tag.Get("json")
					if tag == "-" {
						continue
					}
					tagName := tag
					if idx := strings.IndexByte(tag, ','); idx != -1 {
						tagName = tag[:idx]
					}
					key := tagName
					if key == "" {
						key = field.Name
					}

					if key == req || field.Name == req {
						result[req] = val.Field(i).Interface()
						found = true
						break
					}
				}
				_ = found
			}
		}
		return result, nil

	case reflect.Map:
		if val.IsNil() {
			return make(map[string]any), nil
		}

		capacity := val.Len()
		if len(required) > 0 {
			capacity = len(required)
		}
		result := make(map[string]any, capacity)

		if len(required) > 0 {
			for _, req := range required {
				if exceptMap != nil && exceptMap[req] {
					continue
				}
				mapKey := reflect.ValueOf(req)
				if mapKey.Type().AssignableTo(val.Type().Key()) {
					mVal := val.MapIndex(mapKey)
					if mVal.IsValid() {
						result[req] = mVal.Interface()
						continue
					}
				}

				for _, k := range val.MapKeys() {
					var kStr string
					if k.Kind() == reflect.String {
						kStr = k.String()
					} else {
						kStr = fmt.Sprint(k.Interface())
					}
					if kStr == req {
						result[req] = val.MapIndex(k).Interface()
						break
					}
				}
			}
		} else {
			for _, k := range val.MapKeys() {
				var keyStr string
				if k.Kind() == reflect.String {
					keyStr = k.String()
				} else {
					keyStr = fmt.Sprint(k.Interface())
				}
				if exceptMap != nil && exceptMap[keyStr] {
					continue
				}
				result[keyStr] = val.MapIndex(k).Interface()
			}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("src must be a struct or map (or pointer to them), got %s", val.Kind())
	}
}
