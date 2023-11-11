/*
Package turborpc provides HTTP access to an object's methods.  A server
registers an object, making it visible as a service with the name of the
type of the object.  After registration, methods of the object will be
accessible over HTTP.  A server may register multiple objects (services)
of different types. A server can also generate a JavaScript/TypeScript
client to access service methods.

Service methods can only look schematically like one off

	func (t T) MethodName(ctx context.Context, argument T1) (reply T2, err error)
	func (t T) MethodName(ctx context.Context, argument T1) error
	func (t T) MethodName(ctx context.Context) (reply T2, err error)
	func (t T) MethodName(ctx context.Context) error

where T1 and T2 can be marshaled by encoding/json.

The method's second argument represents the argument provided by the
client; the first return type represents the reply to be returned to
the client.  The method's error value, if non-nil, is passed back to
the client HTTP response with status code 500.  If an error is returned,
the reply will not be sent back to the client.
*/
package turborpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

var (
	ErrReservedServiceName = errors.New("this service name is reserved")
	ErrInvalidService      = errors.New("service must have one or more exported methods")
	ErrServiceRegistered   = errors.New("service name already registered")
)

var (
	errServiceNotFound = errors.New("could not find service")
	errMethodNotFound  = errors.New("could not find method")
	errNoService       = errors.New("no service specified")
	errNoMethod        = errors.New("no method specified")
)

var (
	nullJSON = []byte("null")
)

// A ServerOption is an option for a Server.
type ServerOption func(*Server)

// WithServerJavaScriptClient makes the server serve a JavaScript client on GET
// requests.
func WithServerJavaScriptClient() ServerOption {
	return func(r *Server) {
		r.serveClient = newJavaScriptClient()
	}
}

// Server represents an RPC Server.
type Server struct {
	services    map[string]*service
	serveClient clientGenerator
}

// NewServer returns a new Server with options applied.
func NewServer(options ...ServerOption) *Server {
	rpc := &Server{
		services: make(map[string]*service),
	}

	for _, o := range options {
		o(rpc)
	}

	return rpc
}

func findServiceName(typ reflect.Type) string {
	if typ.Kind() == reflect.Pointer {
		return typ.Elem().Name()
	}

	return typ.Name()
}

func validServiceType(typ reflect.Type) bool {
	return typ.NumMethod() > 0
}

// Register publishes in the server the set of methods of the
// receiver value that are on one of the following forms:
//
//	func (t T) MethodName(ctx context.Context, argument T1) (reply T2, err error)
//	func (t T) MethodName(ctx context.Context, argument T1) error
//	func (t T) MethodName(ctx context.Context) (reply T2, err error)
//	func (t T) MethodName(ctx context.Context) error
//
// where T1 and T2 can be marshaled by encoding/json.
func (rpc *Server) Register(rcvr any) error {
	return rpc.RegisterName(findServiceName(reflect.TypeOf(rcvr)), rcvr)
}

// RegisterName is like Register but uses the provided name for the service
// instead of inferring it from the receiver's type.
func (rpc *Server) RegisterName(name string, r any) error {
	if name == defaultRPCClassName {
		return fmt.Errorf("%s: %w", name, ErrReservedServiceName)
	}

	if _, ok := rpc.services[name]; ok {
		return fmt.Errorf("%s: %w", name, ErrServiceRegistered)
	}

	typ := reflect.TypeOf(r)

	if !validServiceType(typ) {
		return ErrInvalidService
	}

	rpc.services[name] = newService(name, typ, reflect.ValueOf(r))

	return nil
}

func (rpc *Server) call(ctx context.Context, service string, method string, input []byte) ([]byte, error) {
	s, ok := rpc.services[service]

	if !ok {
		return nil, fmt.Errorf("service %q: %w", service, errServiceNotFound)
	}

	m, ok := s.methods[method]

	if !ok {
		return nil, fmt.Errorf("method %q: %w", method, errMethodNotFound)
	}

	return m.invoke(ctx, input)
}

type errorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func httpError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	buf, _ := json.Marshal(errorResponse{
		Status:  code,
		Message: err.Error(),
	})

	w.Write(buf)
}

type outputResponse struct {
	Output json.RawMessage `json:"output"`
}

func httpOK(w http.ResponseWriter, output []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	buf, _ := json.Marshal(outputResponse{
		Output: output,
	})

	w.Write(buf)
}

// ServeHTTP implements an http.Handler that answers RPC requests.
func (rpc *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rpc.serveClient != nil && r.Method == http.MethodGet {
		sourceClient := rpc.serveClient.GenerateClient(rpc.metadata())
		w.Header().Set("Content-Type", sourceClient.ContentType)
		w.Write([]byte(sourceClient.SourceCode))
		return
	}

	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	service := r.URL.Query().Get("service")
	if service == "" {
		httpError(w, http.StatusBadRequest, errNoService)
		return
	}

	method := r.URL.Query().Get("method")
	if method == "" {
		httpError(w, http.StatusBadRequest, errNoMethod)
		return
	}

	input, err := io.ReadAll(r.Body)

	if err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}

	buf, err := rpc.call(r.Context(), service, method, input)

	if err != nil {
		switch {
		case errors.Is(err, errServiceNotFound) || errors.Is(err, errMethodNotFound):
			httpError(w, http.StatusNotFound, err)
		case errors.Is(err, errEncodingOutput):
			httpError(w, http.StatusInternalServerError, err)
		default:
			httpError(w, http.StatusBadRequest, err)
		}

		return
	}

	if buf == nil {
		httpOK(w, nullJSON)
		return
	}

	httpOK(w, buf)
}