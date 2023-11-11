package turborpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

var (
	errNoInput        = errors.New("no input")
	errDecodingInput  = errors.New("decoding input")
	errEncodingOutput = errors.New("encoding output")
)

type method struct {
	name   string
	fn     reflect.Value
	input  reflect.Type
	output reflect.Type
}

func newMethod(m reflect.Method, fn reflect.Value) *method {
	var input, output reflect.Type

	if m.Type.NumIn() == 3 {
		input = m.Type.In(2)
	}

	if m.Type.NumOut() == 2 {
		output = m.Type.Out(0)
	}

	return &method{
		name:   m.Name,
		fn:     fn,
		input:  input,
		output: output,
	}
}

func (m *method) decodeInput(input []byte) (argv reflect.Value, err error) {
	if len(input) == 0 {
		return argv, errNoInput
	}

	argIsValue := false
	if m.input.Kind() == reflect.Pointer {
		argv = reflect.New(m.input.Elem())
	} else {
		argv = reflect.New(m.input)
		argIsValue = true
	}

	if err := json.Unmarshal(input, argv.Interface()); err != nil {
		return argv, err
	}

	if argIsValue {
		argv = argv.Elem()
	}

	return argv, nil
}

func (m *method) invoke(ctx context.Context, bs []byte) ([]byte, error) {
	var outputs []reflect.Value

	if m.input == nil {
		outputs = m.fn.Call([]reflect.Value{reflect.ValueOf(ctx)})
	} else {
		input, err := m.decodeInput(bs)

		if err != nil {
			return nil, fmt.Errorf("%w: %w", errDecodingInput, err)
		}

		outputs = m.fn.Call([]reflect.Value{reflect.ValueOf(ctx), input})
	}

	var resp, errv reflect.Value
	if m.output == nil {
		errv = outputs[0]
	} else {
		resp = outputs[0]
		errv = outputs[1]
	}

	if !errv.IsNil() {
		return nil, errv.Interface().(error)
	}

	if !resp.IsValid() {
		return nil, nil
	}

	buf, err := json.Marshal(resp.Interface())

	if err != nil {
		return nil, fmt.Errorf("%w: %w", errEncodingOutput, err)
	}

	return buf, nil
}
