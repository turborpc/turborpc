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
	ErrMethodErrored       = errors.New("method errored")
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

// WithErrorFilter sets the error filter function for the server.
// It can be used to modify errors returned by the server.
// The server will return ErrMethodErrored if the filter returns nil.
// By default no error filter is set and errors are returned as is from the
// server.
func WithErrorFilter(filter func(err error) error) ServerOption {
	return func(r *Server) {
		r.errorFilter = filter
	}
}

func makeMethodLogger(printf func(format string, a ...any) (n int, err error)) func(service, method string) {
	return func(service, method string) {
		printf("TurboRPC ~ %s::%s\n", service, method)
	}
}

// WithNoMethodLogger disables logging of methods when registering services.
func WithNoMethodLogger() ServerOption {
	return func(r *Server) {
		r.methodLogger = nil
	}
}

// Server represents an RPC Server.
type Server struct {
	errorFilter  func(err error) error
	methodLogger func(service, method string)
	services     map[string]*service
	serveClient  clientGenerator
}

// NewServer returns a new Server with options applied.
func NewServer(options ...ServerOption) *Server {
	rpc := &Server{
		errorFilter:  nil,
		methodLogger: makeMethodLogger(fmt.Printf),
		services:     make(map[string]*service),
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

	rpc.services[name] = newService(name, typ, reflect.ValueOf(r), rpc.methodLogger)

	return nil
}

func (rpc *Server) call(ctx context.Context, service string, method string, input []byte) ([]byte, error) {
	s, ok := rpc.services[service]

	if !ok {
		return nil, fmt.Errorf("%w %q", errServiceNotFound, service)
	}

	m, ok := s.methods[method]

	if !ok {
		return nil, fmt.Errorf("%w %q", errMethodNotFound, method)
	}

	return m.invoke(ctx, input)
}

type errorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// Error replies to the request with the specified error message and HTTP code.
// The format of the reply is the one expected by TurboRPC clients. It does not
// otherwise end the request; the caller should ensure no further writes are
// done to w.
func Error(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	buf, _ := json.Marshal(errorResponse{
		Status:  code,
		Message: error,
	})

	w.Write(buf)
}

func httpError(w http.ResponseWriter, code int, err error) {
	if err == nil {
		Error(w, ErrMethodErrored.Error(), code)
		return
	}

	Error(w, err.Error(), code)
}

type outputResponse struct {
	Output json.RawMessage `json:"output"`
}

func httpOK(w http.ResponseWriter, output json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
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
			if rpc.errorFilter != nil {
				httpError(w, http.StatusBadRequest, rpc.errorFilter(err))
			} else {
				httpError(w, http.StatusBadRequest, err)
			}
		}

		return
	}

	if buf == nil {
		httpOK(w, nullJSON)
		return
	}

	httpOK(w, buf)
}
