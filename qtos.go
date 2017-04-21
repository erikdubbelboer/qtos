package qtos

import (
	"errors"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrMustBeStruct  = errors.New("Must be a struct")
	ErrMustBePointer = errors.New("v must be a pointer")

	// StructTag is the struct tag key being used.
	// Assigning to this variable is not thread safe.
	StructTag = "query"

	mapRegexp = regexp.MustCompile(`\[[a-z]+\]$`)
)

// Unmarshal parses the url values and stores the result in the value pointed to by v
// For examples and supported formats see the tests.
func Unmarshal(values url.Values, v interface{}) error {
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	if val.Kind() != reflect.Ptr || val.IsNil() {
		return ErrMustBePointer
	} else {
		typ = typ.Elem()
		val = val.Elem()
	}

	return bindStruct(typ, val, "", values)
}

func bindStruct(typ reflect.Type, val reflect.Value, prefix string, values url.Values) error {
	if typ.Kind() != reflect.Struct {
		return ErrMustBeStruct
	}

	mapping := make(map[string]int)

	for i := 0; i < typ.NumField(); i++ {
		styp := typ.Field(i)

		valueName := styp.Tag.Get(StructTag)

		if valueName == "" {
			valueName = styp.Name
		}

		mapping[valueName] = i
	}

	for name, value := range values {
		// If we are inside a stuct we should make sure we only bind values for that struct.
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// Trim the struct prefix.
		name = strings.TrimPrefix(name, prefix)

		// Is it a struct?
		if strings.Count(name, ".") > 0 {
			parts := strings.SplitN(name, ".", 2)
			// parts[0] will be the name of the value in the current struct.
			i, ok := mapping[parts[0]]
			if !ok {
				continue
			}
			// Recurse into the new struct with the next prefix.
			if err := bindStruct(typ.Field(i).Type, val.Field(i), prefix+parts[0]+".", values); err != nil {
				return err
			}
			continue
		}

		// Ignore any [] at the end as our values are always a slice of values anyways.
		if strings.HasSuffix(name, "[]") {
			name = name[:len(name)-2]
		}

		// Is it a map? If so get the key for the map.
		key := mapRegexp.FindString(name)
		if key != "" {
			// Remove the map key from the name.
			name = strings.TrimSuffix(name, key)
			key = key[1 : len(key)-1]
		}

		i, ok := mapping[name]
		if !ok {
			continue
		}

		styp := typ.Field(i)
		sval := val.Field(i)

		if err := assign(styp, sval, key, value); err != nil {
			return err
		}
	}

	return nil
}

func assign(typ reflect.StructField, val reflect.Value, key string, value []string) error {
	switch typ.Type.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		if i, err := strconv.ParseInt(value[0], 10, 64); err != nil {
			return err
		} else {
			val.SetInt(i)
		}
	case reflect.String:
		// Just use the zero'th value only. In theory we could also return an error if there are multiple values.
		val.SetString(value[0])
	case reflect.Slice:
		// TODO: right now this will only support slices of strings because
		// value is a slice of strings. If you want to support other slices
		// you need to look at the type of the slice value and convert value
		// to a slice of this type.
		val.Set(reflect.ValueOf(value))
	case reflect.Map:
		if val.IsNil() {
			// Create the map if it doesn't exist yet.
			val.Set(reflect.MakeMap(typ.Type))
		}
		val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	return nil
}
