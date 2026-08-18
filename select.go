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

	exceptMap := make(map[string]bool, len(ex))
	for _, e := range ex {
		exceptMap[e] = true
	}

	result := make(map[string]any)

	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		fieldValues := make(map[string]reflect.Value)
		var allKeys []string

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" { // Skip unexported fields
				continue
			}

			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}

			tagName := strings.Split(tag, ",")[0]
			key := tagName
			if key == "" {
				key = field.Name
			}

			fieldValues[key] = val.Field(i)
			fieldValues[field.Name] = val.Field(i)
			allKeys = append(allKeys, key)
		}

		targetKeys := required
		if len(targetKeys) == 0 {
			targetKeys = allKeys
		}

		for _, req := range targetKeys {
			if exceptMap[req] {
				continue
			}
			if fVal, ok := fieldValues[req]; ok {
				result[req] = fVal.Interface()
			}
		}

	case reflect.Map:
		if val.IsNil() {
			return result, nil
		}

		if len(required) > 0 {
			for _, req := range required {
				if exceptMap[req] {
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
					if fmt.Sprint(k.Interface()) == req {
						result[req] = val.MapIndex(k).Interface()
						break
					}
				}
			}
		} else {
			for _, k := range val.MapKeys() {
				keyStr := fmt.Sprint(k.Interface())
				if exceptMap[keyStr] {
					continue
				}
				result[keyStr] = val.MapIndex(k).Interface()
			}
		}

	default:
		return nil, fmt.Errorf("src must be a struct or map (or pointer to them), got %s", val.Kind())
	}

	return result, nil
}
