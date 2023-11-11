package turborpc

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/olahol/tsreflect"
)

func TestDate(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		var x Date

		b, err := json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, fmt.Sprintf(`"%s(%s)"`, datePrefix, "0001-01-01T00:00:00Z"), string(b))
	})

	t.Run("type", func(t *testing.T) {
		var x Date

		typ := x.TypeScriptType(tsreflect.New(), false)

		assertEqual(t, "Date", typ)
	})
}

func TestNonNullSlice(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		var x NonNullSlice[string]

		b, err := json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, "[]", string(b))

		x = append(x, "test")

		b, err = json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, `["test"]`, string(b))
	})

	t.Run("type", func(t *testing.T) {
		var x NonNullSlice[int]

		typ := x.TypeScriptType(tsreflect.New(), false)

		assertEqual(t, "number[]", typ)
	})

	t.Run("byte marshal", func(t *testing.T) {
		var x NonNullSlice[byte]

		b, err := json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, `""`, string(b))

		x = append(x, 255)

		b, err = json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, `"/w=="`, string(b))

	})

	t.Run("byte type", func(t *testing.T) {
		var x NonNullSlice[byte]

		typ := x.TypeScriptType(tsreflect.New(), false)

		assertEqual(t, "string", typ)
	})
}

func TestNonNullMap(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		var x NonNullMap[int, int]

		b, err := json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, "{}", string(b))

		x = NonNullMap[int, int]{
			10: 11,
		}

		b, err = json.Marshal(x)

		assertNoError(t, err)
		assertEqual(t, `{"10":11}`, string(b))
	})

	t.Run("type", func(t *testing.T) {
		var x NonNullMap[int, int]

		typ := x.TypeScriptType(tsreflect.New(), false)

		assertEqual(t, "{ [key: number]: number }", typ)
	})
}
