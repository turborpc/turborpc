package turborpc

import (
	"context"
	"reflect"
)

var (
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()
)

type service struct {
	name    string
	version string
	typ     reflect.Type
	value   reflect.Value
	methods map[string]*method
}

func newService(name string, typ reflect.Type, value reflect.Value, logger func(service, method string)) *service {
	s := &service{
		name:    name,
		typ:     typ,
		value:   value,
		methods: make(map[string]*method),
	}

	for i := 0; i < s.typ.NumMethod(); i++ {
		m := s.typ.Method(i)

		if !isSuitableMethod(m) {
			continue
		}

		s.methods[m.Name] = newMethod(m, s.value.Method(i))

		if logger != nil {
			logger(s.name, m.Name)
		}
	}

	s.version = calculateServiceVersion(s.metadata())

	return s
}

func isSuitableMethod(m reflect.Method) bool {
	correctInputsAndOutputs := m.Type.NumIn() > 1 && m.Type.NumIn() < 4 && m.Type.NumOut() > 0 && m.Type.NumOut() < 3

	if !m.IsExported() || !correctInputsAndOutputs || m.Type.In(1) != typeOfContext || (m.Type.NumOut() == 1 && m.Type.Out(0) != typeOfError) || (m.Type.NumOut() == 2 && m.Type.Out(1) != typeOfError) {
		return false
	}

	return true
}
