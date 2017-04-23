package qtos

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	// StructTag is the struct tag key being used.
	// Assigning to this variable is not thread safe.
	StructTag = "query"

	fieldRegexp  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	indexRegexp  = regexp.MustCompile(`^\[[0-9]+\]`)
	mapKeyRegexp = regexp.MustCompile(`^\[[^\]]+\]`)
)

// Unmarshal parses the url values and stores the result in the value pointed to by v
// For examples and supported formats see the tests.
func Unmarshal(values url.Values, v interface{}) error {
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("v must be a pointer")
	} else {
		typ = typ.Elem()
		val = val.Elem()
	}

	for name, value := range values {
		if err := bind(typ, val, "", name, value); err != nil {
			return err
		}
	}

	return nil
}

// bind is called recursively while parsing the value name.
// base contains the parsed part so far and is only used to product nice error messages.
func bind(typ reflect.Type, val reflect.Value, base, name string, value []string) error {
	// If the name is empty it means we should assign value to val.
	if name == "" {
		if len(value) > 1 {
			return fmt.Errorf("expected only one value for %s got %v", base, value)
		}
		if v, err := getValue(typ, value[0]); err != nil {
			return err
		} else {
			val.Set(v)
			return nil
		}
	}

	// We can ignore leading dots.
	if name[0] == '.' {
		base = base + "."
		name = name[1:]
	}

	// Is it a struct field?
	if field := fieldRegexp.FindString(name); field != "" {
		if typ.Kind() != reflect.Struct {
			return fmt.Errorf("expected a struct for %s got %v", base, typ)
		}

		i, ok := getStructField(typ, field)
		if !ok {
			// The struct doesn't have any field with this name.
			return nil
		}

		return bind(typ.Field(i).Type, val.Field(i), base+name[:len(field)], name[len(field):], value)
	}

	// Is it a slice?
	if strings.HasPrefix(name, "[]") {
		if typ.Kind() != reflect.Slice {
			return fmt.Errorf("expected a slice for %s got %v", base, typ)
		}

		if len(name) != 2 {
			return fmt.Errorf("[] can only be at the end of the key")
		}

		for _, v := range value {
			if vv, err := getValue(typ.Elem(), v); err != nil {
				return err
			} else {
				val.Set(reflect.Append(val, vv))
			}
		}
		return nil
	}

	// Only try this with a slice destination, otherwise it might be an integer map key.
	if typ.Kind() == reflect.Slice {
		// Is it a slice index?
		if indexStr := indexRegexp.FindString(name); indexStr != "" {
			index, _ := strconv.Atoi(indexStr[1 : len(indexStr)-1])

			// Do we need to create the slice?
			if val.IsNil() {
				val.Set(reflect.MakeSlice(val.Type(), index+1, index+1))
			} else if val.Len() < index+1 {
				// The slice isn't big enough.
				if val.Cap() < index+1 {
					n := reflect.MakeSlice(val.Type(), index+1, index+1)
					reflect.Copy(n, val)
					val.Set(n)
				} else {
					val.SetLen(index + 1)
				}
			}

			t := typ.Elem()
			v := reflect.Indirect(reflect.New(t))

			if err := bind(t, v, base+name[:len(indexStr)], name[len(indexStr):], value); err != nil {
				return err
			} else if mv, err := mergeValues(val.Index(index), v); err != nil {
				return err
			} else {
				val.Index(index).Set(mv)
				return nil
			}
		}
	}

	// Is it a map key?
	if key := mapKeyRegexp.FindString(name); key != "" {
		if typ.Kind() != reflect.Map {
			return fmt.Errorf("expected a map for %s got %v", base, typ)
		}

		if val.IsNil() {
			// Create the map if it doesn't exist yet.
			val.Set(reflect.MakeMap(typ))
		}

		t := typ.Elem()
		v := reflect.Indirect(reflect.New(t))

		if err := bind(t, v, base+name[:len(key)], name[len(key):], value); err != nil {
			return err
		} else {
			if k, err := getValue(typ.Key(), key[1:len(key)-1]); err != nil {
				return err
			} else if mv, err := mergeValues(val.MapIndex(k), v); err != nil {
				return err
			} else {
				val.SetMapIndex(k, mv)
				return nil
			}
		}
	}

	return fmt.Errorf("unknown format %s in %s", name, base+name)
}

func mergeValues(a, b reflect.Value) (reflect.Value, error) {
	if !a.IsValid() {
		return b, nil
	}

	if a.Kind() != b.Kind() {
		return a, fmt.Errorf("can not merge %v and %v", a.Type(), b.Type())
	}

	switch a.Kind() {
	case reflect.Slice:
		if a.IsNil() {
			return b, nil
		}

		// Make sure a is always the longest slice.
		if b.Len() > a.Len() {
			a, b = b, a
		}

		for i := 0; i < b.Len(); i++ {
			a.Index(i).Set(b.Index(i))
		}

		return a, nil
	case reflect.Map:
		if a.IsNil() {
			return b, nil
		}

		for _, k := range b.MapKeys() {
			a.SetMapIndex(k, b.MapIndex(k))
		}

		return a, nil
	default:
		// For most values like string we just return the second value
		// so it will always overwrite the first.
		return b, nil
	}
}

// getValue returns value as a reflect.Value of type typ.
func getValue(typ reflect.Type, value string) (reflect.Value, error) {
	switch typ.Kind() {
	case reflect.String:
		return reflect.ValueOf(value), nil
	case reflect.Int, reflect.Int32, reflect.Int64:
		if i, err := strconv.ParseInt(value, 10, 64); err != nil {
			return reflect.Zero(typ), err
		} else {
			v := reflect.Indirect(reflect.New(typ))
			v.SetInt(i)
			return v, nil
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(value, 64); err != nil {
			return reflect.Zero(typ), err
		} else {
			v := reflect.Indirect(reflect.New(typ))
			v.SetFloat(f)
			return v, nil
		}
	case reflect.Bool:
		if b, err := strconv.ParseBool(value); err != nil {
			return reflect.Zero(typ), err
		} else {
			v := reflect.Indirect(reflect.New(typ))
			v.SetBool(b)
			return v, nil
		}
	case reflect.Interface:
		return reflect.ValueOf(value), nil
	default:
		return reflect.Zero(typ), fmt.Errorf("unsupported type %v", typ)
	}
}

// getStructField returns the field index in typ of the field with struct
// tag name.
func getStructField(typ reflect.Type, name string) (int, bool) {
	// TODO: In theory we could add a global case that caches this mapping
	// based on typ.Name(). It would require a mutex so some benchmarking
	// is required to see if this actually improves the speed.
	mapping := make(map[string]int)

	for i := 0; i < typ.NumField(); i++ {
		styp := typ.Field(i)

		valueName := styp.Tag.Get(StructTag)

		if valueName == "" {
			valueName = styp.Name
		}

		mapping[valueName] = i
	}

	i, ok := mapping[name]
	return i, ok
}
