package qtos

import (
	"net/url"
	"reflect"
	"testing"
)

type (
	Search struct {
		Query  string                 `query:"q"`
		Filter map[string]interface{} `query:"filter"`
		Fields []string               `query:"fields"`
		Paging Paging                 `query:"paging"`
	}

	Paging struct {
		Offset int64               `query:"offset"`
		Size   int32               `query:"size"`
		Foo    map[string][]string `query:"foo"`
		Nested Boo                 `query:"nested"`
	}

	Boo struct {
		Bar int `query:"bar"`
	}
)

func TestUnmarshal(t *testing.T) {
	inputs := map[string]Search{
		"q=Honda&filter[categories][]=Motor&filter[categories][]=Mobil&fields[]=foo&fields[]=bar&paging.offset=2&paging.size=100&paging.foo[bar][]=bla&paging.nested.bar=1337": Search{
			Query: "Honda",
			Filter: map[string]interface{}{
				"categories": []string{"Motor", "Mobil"},
			},
			Fields: []string{"foo", "bar"},
			Paging: Paging{
				Offset: 2,
				Size:   100,
				Foo: map[string][]string{
					"bar": []string{"bla"},
				},
				Nested: Boo{
					Bar: 1337,
				},
			},
		},
	}

	for qs, expected := range inputs {
		t.Run(qs, func(t *testing.T) {
			values, err := url.ParseQuery(qs)
			if err != nil {
				panic(err)
			}

			x := Search{}
			if err := Unmarshal(values, &x); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expected, x) {
				t.Fatalf("expected\n%#v\ngot\n%#v", expected, x)
			}
		})
	}
}
