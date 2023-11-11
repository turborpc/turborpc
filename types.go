package turborpc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/olahol/tsreflect"
)

// datePrefix string used to prefix turborpc.Date objects when marshaled to JSON.
const datePrefix = `__turborpc.Date`

var typeOfByteSlice = reflect.TypeOf([]byte{})

var (
	emptyJSONString = []byte(`""`)
	emptyJSONArray  = []byte("[]")
	emptyJSONObject = []byte("{}")
)

// A Date is a time.Time object that can be "revived" into
// a JavaScript "Date" object by a client.
type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
	bs, err := time.Time(d).MarshalText()

	return []byte(fmt.Sprintf(`"%s(%s)"`, datePrefix, bs)), err
}

func (Date) TypeScriptType(g *tsreflect.Generator, optional bool) string {
	return "Date"
}

// A NonNullSlice is a slice where the zero value (nil) is marshaled
// into "[]" instead of "null". Unlike regular slices its TypeScript
// type is "T[]" not "T[] | null". A nil byte slice ([]byte) marshals into an
// empty string.
type NonNullSlice[T any] []T

func (nns NonNullSlice[T]) MarshalJSON() ([]byte, error) {
	if nns == nil {
		if reflect.TypeOf([]T(nns)) == typeOfByteSlice {
			return emptyJSONString, nil
		}

		return emptyJSONArray, nil
	}

	return json.Marshal([]T(nns))
}

func (nns NonNullSlice[T]) TypeScriptType(g *tsreflect.Generator, optional bool) string {
	if reflect.TypeOf([]T(nns)) == typeOfByteSlice {
		return "string"
	}

	typ := reflect.TypeOf(nns).Elem()

	return fmt.Sprintf("%s[]", g.TypeOf(typ))
}

// A NonNullMap is a map where the zero value (nil) is marshaled
// into "{}" instead of "null". Unlike regular maps its TypeScript
// type is "{[key: K]: V}" not "{[key: K]: V} | null".
type NonNullMap[K comparable, V any] map[K]V

func (nnm NonNullMap[K, V]) MarshalJSON() ([]byte, error) {
	if nnm == nil {
		return emptyJSONObject, nil
	}

	return json.Marshal(map[K]V(nnm))
}

func (nnm NonNullMap[K, V]) TypeScriptType(g *tsreflect.Generator, optional bool) string {
	typ := reflect.TypeOf(nnm)
	return fmt.Sprintf("{ [key: %s]: %s }", g.TypeOf(typ.Key()), g.TypeOf(typ.Elem()))
}
