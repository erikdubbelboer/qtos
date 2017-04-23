package qtos

import (
	"net/url"
	"reflect"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	type subsub struct {
		Int int `query:"int"`
	}

	type sub struct {
		Int         int         `query:"int"`
		Interface   interface{} `query:"interface"`
		SubSub      subsub      `query:"subsub"`
		SliceString []string    `query:"slicestring"`
	}

	type base struct {
		String               string                `query:"string"`
		Int                  int                   `query:"int"`
		Float                float64               `query:"float"`
		Bool                 bool                  `query:"bool"`
		Interface            interface{}           `query:"interface"`
		MapStringInt         map[string]int        `query:"mapstringint"`
		MapIntString         map[int]string        `query:"mapintstring"`
		MapIntInterface      map[int]interface{}   `query:"mapintinterface"`
		SliceString          []string              `query:"slicestring"`
		SliceInt             []int                 `query:"sliceint"`
		SliceInterface       []interface{}         `query:"sliceinterface"`
		MapStringSliceString map[string][]string   `query:"mapstringslicestring"`
		MapIntSliceInt       map[int][]int         `query:"mapintsliceint"`
		MapIntSliceInterface map[int][]interface{} `query:"mapintsliceinterface"`
		Sub                  sub                   `query:"sub"`
		MapStringSub         map[string]sub        `query:"mapstringsub"`
		SliceMapIntInt       []map[int]int         `query:"slicemapintint"`
	}

	inputs := map[string]interface{}{
		"string=test": base{
			String: "test",
		},
		"int=2": base{
			Int: 2,
		},
		"float=2.3": base{
			Float: 2.3,
		},
		"bool=true": base{
			Bool: true,
		},
		"interface=test": base{
			Interface: "test",
		},
		"interface=2": base{
			Interface: "2",
		},
		"mapstringint[test]=2": base{
			MapStringInt: map[string]int{
				"test": 2,
			},
		},
		"mapintstring[2]=test": base{
			MapIntString: map[int]string{
				2: "test",
			},
		},
		"mapintinterface[2]=test": base{
			MapIntInterface: map[int]interface{}{
				2: "test",
			},
		},
		"slicestring[]=foo&slicestring[]=bar": base{
			SliceString: []string{"foo", "bar"},
		},
		"slicestring[1]=foo&slicestring[0]=bar": base{
			SliceString: []string{"bar", "foo"},
		},
		"sliceint[]=2&sliceint[]=3": base{
			SliceInt: []int{2, 3},
		},
		"sliceinterface[]=foo&sliceinterface[]=bar": base{
			SliceInterface: []interface{}{"foo", "bar"},
		},
		"mapstringslicestring[test][]=foo&mapstringslicestring[test][]=bar&mapstringslicestring[foo][0]=bar": base{
			MapStringSliceString: map[string][]string{
				"test": []string{"foo", "bar"},
				"foo":  []string{"bar"},
			},
		},
		"mapstringslicestring[test][1]=foo&mapstringslicestring[test][0]=bar": base{
			MapStringSliceString: map[string][]string{
				"test": []string{"bar", "foo"},
			},
		},
		"mapintsliceint[2][]=3&mapintsliceint[4][]=5": base{
			MapIntSliceInt: map[int][]int{
				2: []int{3},
				4: []int{5},
			},
		},
		"mapintsliceinterface[2][]=foo&mapintsliceinterface[2][]=bar": base{
			MapIntSliceInterface: map[int][]interface{}{
				2: []interface{}{"foo", "bar"},
			},
		},
		"sub.int=2": base{
			Sub: sub{
				Int: 2,
			},
		},
		"sub.interface=2": base{
			Sub: sub{
				Interface: "2",
			},
		},
		"sub.subsub.int=2": base{
			Sub: sub{
				SubSub: subsub{
					Int: 2,
				},
			},
		},
		"sub.slicestring[]=foo&sub.slicestring[]=bar": base{
			Sub: sub{
				SliceString: []string{"foo", "bar"},
			},
		},
		"mapstringsub[foo].int=2": base{
			MapStringSub: map[string]sub{
				"foo": sub{
					Int: 2,
				},
			},
		},
		"slicemapintint[0][1]=2&slicemapintint[0][3]=4": base{
			SliceMapIntInt: []map[int]int{
				map[int]int{
					1: 2,
					3: 4,
				},
			},
		},

		// Test some none-struct values.
		"[]=foo&[]=bar": []string{"foo", "bar"},
		"[]=2&[]=3":     []int{2, 3},
		"[1]=3&[0]=2":   []int{2, 3},
		"[foo]=bar":     map[string]string{"foo": "bar"},
		"[2]=3":         map[int]int{2: 3},
		"[4]=5":         map[string]int{"4": 5},
	}

	for qs, expected := range inputs {
		t.Run(qs, func(t *testing.T) {
			values, err := url.ParseQuery(qs)
			if err != nil {
				t.Fatal(err)
			}

			// Make an empty value of the same type as expected.
			x := reflect.New(reflect.TypeOf(expected))
			if err := Unmarshal(values, x.Interface()); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(expected, reflect.Indirect(x).Interface()) {
				t.Fatalf("expected\n%#v\ngot\n%#v", expected, x)
			}
		})
	}
}

func TestError(t *testing.T) {
	type sub struct {
		Int int `query:"int"`
	}

	type base struct {
		SliceSub       []sub         `query:"slicesub"`
		SliceMapIntInt []map[int]int `query:"slicemapintint"`
	}

	inputs := map[string]interface{}{
		"slicesub[].int=2&slicesub[].int=3":           base{},
		"slicemapintint[][2]=3&slicemapintint[][2]=4": base{},
	}

	for qs, expected := range inputs {
		t.Run(qs, func(t *testing.T) {
			values, err := url.ParseQuery(qs)
			if err != nil {
				t.Fatal(err)
			}

			// Make an empty value of the same type as expected.
			x := reflect.New(reflect.TypeOf(expected))
			if err := Unmarshal(values, x.Interface()); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
