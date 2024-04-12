package turborpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestService1 struct{}

func (c *TestService1) One(context.Context) error {
	return nil
}

func (c *TestService1) Two(context.Context, int) error {
	return nil
}

func (c *TestService1) Three(context.Context, int) (int, error) {
	return 3, nil
}

func (c *TestService1) Four(ctx context.Context, arg struct{ a int }) (int, error) {
	return 4, nil
}

func (c *TestService1) Error(ctx context.Context, msg string) error {
	return errors.New(msg)
}

func (c *TestService1) Pointer(ctx context.Context, msg *string) (string, error) {
	return *msg, nil
}

type TestService2 struct{}

func (c *TestService2) One(context.Context) error {
	return nil
}

func (c *TestService2) Two(context.Context, int) error {
	return nil
}

func (c *TestService2) Three(context.Context, int) (int, error) {
	return 3, nil
}

func (c *TestService2) Four(ctx context.Context, arg struct{ a int }) (int, error) {
	return 4, nil
}

type TestServiceEcho struct{}

func (c *TestServiceEcho) Echo(ctx context.Context, input any) (any, error) {
	return input, nil
}

var errTest = errors.New("test")

func (c *TestServiceEcho) Error(ctx context.Context, input any) error {
	return errTest
}

type TestServiceFunction struct{}

func (c *TestServiceFunction) Test(ctx context.Context) (func(), error) {
	return func() {}, nil
}

func (c *TestServiceFunction) Wrong(ctx context.Context, a int, b int) error {
	return nil
}

func (c *TestServiceFunction) Invalid(ctx context.Context, a int) error {
	return nil
}

func MustMarshalJSON(input any) io.Reader {
	b, err := json.Marshal(input)

	if err != nil {
		panic(err)
	}

	return bytes.NewReader(b)
}

func MustUnmarshalJSON[T any](r io.Reader) (t T) {
	b, err := io.ReadAll(r)

	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &t)

	if err != nil {
		panic(err)
	}

	return
}

func callRpc[A any, B any](rpc *Server, service string, method string, input B) A {
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/?service=%s&method=%s", service, method), MustMarshalJSON(input))
	w := httptest.NewRecorder()

	rpc.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	o := MustUnmarshalJSON[struct {
		Output A `json:"output"`
	}](res.Body)

	return o.Output
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func newTestServer(options ...ServerOption) *Server {
	return NewServer(append(options, WithNoMethodLogger())...)
}

func TestServer(t *testing.T) {
	t.Run("string echo", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestServiceEcho{})

		input := "Hello World!"
		output := callRpc[string](rpc, "TestServiceEcho", "Echo", input)

		assertEqual(t, input, output)
	})

	t.Run("pointer echo", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		input := "Hello World!"
		output := callRpc[string](rpc, "TestService1", "Pointer", input)

		assertEqual(t, input, output)
	})

	t.Run("invalid", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestServiceFunction{})

		callRpc[int](rpc, "TestServiceFunction", "Invalid", 1)
	})

	t.Run("struct echo", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestServiceEcho{})

		type S struct {
			A string
			B int
		}

		input := S{
			A: "Hello World!",
			B: 101,
		}
		output := callRpc[S](rpc, "TestServiceEcho", "Echo", input)

		assertEqual(t, input, output)
	})
}

func TestServerErrors(t *testing.T) {
	t.Run("non post method", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		req := httptest.NewRequest(http.MethodGet, "/?service=TestService1&method=NotFound", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("missing service", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		req := httptest.NewRequest(http.MethodPost, "/?method=NotFound", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("missing method", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		req := httptest.NewRequest(http.MethodPost, "/?service=TestService1", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("service not found", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		req := httptest.NewRequest(http.MethodPost, "/?service=NotFound&method=NotFound", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("method not found", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		req := httptest.NewRequest(http.MethodPost, "/?service=TestService1&method=NotFound", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("malformed input", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		req := httptest.NewRequest(http.MethodPost, "/?service=TestService1&method=Two", strings.NewReader("malformed input"))
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("no input", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestServiceEcho{})

		req := httptest.NewRequest(http.MethodPost, "/?service=TestServiceEcho&method=Echo", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("method error", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})

		input := "an error"
		req := httptest.NewRequest(http.MethodPost, "/?service=TestService1&method=Error", strings.NewReader(fmt.Sprintf("%q", input)))
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		assertEqual(t, http.StatusBadRequest, res.StatusCode)

		o := MustUnmarshalJSON[errorResponse](res.Body)

		assertEqual(t, input, o.Message)
	})

	t.Run("register invalid service", func(t *testing.T) {
		rpc := newTestServer()

		type InvalidService struct {
		}

		err := rpc.Register(InvalidService{})

		assertErrorIs(t, ErrInvalidService, err)
	})

	t.Run("register duplicate service", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})
		err := rpc.Register(&TestService1{})

		assertErrorIs(t, ErrServiceRegistered, err)
	})

	t.Run("register reserved service", func(t *testing.T) {
		rpc := newTestServer()

		err := rpc.RegisterName(defaultRPCClassName, &TestService1{})

		assertErrorIs(t, ErrReservedServiceName, err)
	})

	t.Run("cannot marshal", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestServiceFunction{})

		req := httptest.NewRequest(http.MethodPost, "/?service=TestServiceFunction&method=Test", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusInternalServerError, res.StatusCode)
	})

	t.Run("cannot read", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestServiceFunction{})

		req := httptest.NewRequest(http.MethodPost, "/?service=TestServiceFunction&method=Test", errReader{})
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		_, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, http.StatusInternalServerError, res.StatusCode)
	})

	t.Run("filtered error", func(t *testing.T) {
		e := "filtered error"
		rpc := newTestServer(WithErrorFilter(func(err error) error {
			return errors.New(e)
		}))

		rpc.Register(&TestService1{})

		input := "an error"
		req := httptest.NewRequest(http.MethodPost, "/?service=TestService1&method=Error", strings.NewReader(fmt.Sprintf("%q", input)))
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		assertEqual(t, http.StatusBadRequest, res.StatusCode)

		o := MustUnmarshalJSON[errorResponse](res.Body)

		assertEqual(t, e, o.Message)
	})

	t.Run("filtered error default", func(t *testing.T) {
		rpc := newTestServer(WithErrorFilter(func(err error) error {
			return nil
		}))

		rpc.Register(&TestService1{})

		input := "an error"
		req := httptest.NewRequest(http.MethodPost, "/?service=TestService1&method=Error", strings.NewReader(fmt.Sprintf("%q", input)))
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		assertEqual(t, http.StatusBadRequest, res.StatusCode)

		o := MustUnmarshalJSON[errorResponse](res.Body)

		assertEqual(t, ErrMethodErrored.Error(), o.Message)
	})
}

func TestServerOptions(t *testing.T) {
	t.Run("serve javascript client", func(t *testing.T) {
		rpc := newTestServer(WithServerJavaScriptClient())

		rpc.Register(&TestService1{})
		rpc.Register(&TestService2{})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		jsClient, err := io.ReadAll(res.Body)

		assertNoError(t, err)

		assertEqual(t, rpc.JavaScriptClient(), string(jsClient))
	})

	t.Run("not serve javascript client", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService1{})
		rpc.Register(&TestService2{})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close()

		assertEqual(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("logger", func(t *testing.T) {
		rpc := newTestServer()

		var called bool
		rpc.methodLogger = makeMethodLogger(func(format string, a ...any) (n int, err error) {
			called = true
			return 0, nil
		})

		rpc.Register(&TestService1{})

		assertEqual(t, called, true)
	})
}
