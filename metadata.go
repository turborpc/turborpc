package turborpc

import (
	"reflect"
	"sort"
)

const defaultRPCClassName = "RPC"

// methodMetadata metadata describing a service method.
type methodMetadata struct {
	Name   string
	Input  reflect.Type
	Output reflect.Type
}

// serviceMetadata metadata describing a server service.
type serviceMetadata struct {
	Name    string
	Methods []methodMetadata
}

// serverMetadata metadata describing a server.
type serverMetadata struct {
	Name     string
	Services []serviceMetadata
}

// types get all method types, both input and output.
func (i serverMetadata) types() []reflect.Type {
	var typs []reflect.Type

	for _, s := range i.Services {
		for _, m := range s.Methods {
			typs = append(typs, m.Input)
			typs = append(typs, m.Output)
		}
	}

	return typs
}

func (m *method) metadata() methodMetadata {
	return methodMetadata{
		Name:   m.name,
		Input:  m.input,
		Output: m.output,
	}
}

func (s *service) metadata() serviceMetadata {
	var ms []methodMetadata
	for _, m := range s.methods {
		ms = append(ms, m.metadata())
	}

	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Name < ms[j].Name
	})

	return serviceMetadata{
		Name:    s.name,
		Methods: ms,
	}
}

// metadata get metadata describing the server.
func (rpc *Server) metadata() serverMetadata {
	var ss []serviceMetadata
	for _, s := range rpc.services {
		ss = append(ss, s.metadata())
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Name < ss[j].Name
	})

	return serverMetadata{
		Name:     defaultRPCClassName,
		Services: ss,
	}
}
