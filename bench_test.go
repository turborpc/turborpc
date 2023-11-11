package turborpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type BenchmarkService struct{}

func (c *BenchmarkService) Echo(ctx context.Context, input any) (any, error) {
	return input, nil
}

func (c *BenchmarkService) Error(ctx context.Context) error {
	return errTest
}

func (c *BenchmarkService) NoInput(ctx context.Context) (int, error) {
	return 1, nil
}

func (c *BenchmarkService) NoOutput(ctx context.Context, input any) error {
	return nil
}

func (c *BenchmarkService) NoInputNoOutput(ctx context.Context) error {
	return nil
}

func BenchmarkEcho(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&BenchmarkService{})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/?service=BenchmarkService&method=Echo", strings.NewReader("1"))

		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		w.Result().Body.Close()
	}
}

func BenchmarkError(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&BenchmarkService{})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/?service=BenchmarkService&method=Error", nil)

		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		w.Result().Body.Close()
	}
}

func BenchmarkNoInput(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&BenchmarkService{})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/?service=BenchmarkService&method=NoInput", nil)

		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		w.Result().Body.Close()
	}
}

func BenchmarkNoOutput(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&BenchmarkService{})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/?service=BenchmarkService&method=NoOutput", strings.NewReader("0"))

		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		w.Result().Body.Close()
	}
}

func BenchmarkNoInputNoOutput(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&BenchmarkService{})

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/?service=BenchmarkService&method=NoInputNoOutput", nil)

		w := httptest.NewRecorder()

		rpc.ServeHTTP(w, req)

		w.Result().Body.Close()
	}
}

func BenchmarkJavaScriptClient(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&TestService1{})
	rpc.Register(&TestService2{})

	for i := 0; i < b.N; i++ {
		_ = rpc.JavaScriptClient()
	}
}

func BenchmarkTypeScriptClient(b *testing.B) {
	rpc := NewServer()
	rpc.Register(&TestService1{})
	rpc.Register(&TestService2{})

	for i := 0; i < b.N; i++ {
		_ = rpc.TypeScriptClient()
	}
}
