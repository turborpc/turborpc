package turborpc

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var runClientTests = os.Getenv("RUN_CLIENT_TESTS") == "yes"

func testClientStability(t *testing.T, client clientGenerator) {
	rpc := newTestServer()

	rpc.Register(&TestService2{})
	rpc.Register(&TestService1{})

	ref := rpc.clientSourceCode(client)

	for i := 0; i < 50; i++ {
		assertEqual(t, ref, rpc.clientSourceCode(client))
	}
}

func execWithOutput(name string, args ...string) (string, error) {
	stdout, err := exec.Command(name, args...).Output()

	if err != nil {
		e, ok := err.(*exec.ExitError)

		if !ok {
			return "", err
		}

		if len(e.Stderr) > 0 {
			return "", errors.New(string(e.Stderr))
		}

		if len(stdout) > 0 {
			return "", errors.New(string(stdout))
		}

		return "", err
	}

	return strings.TrimSpace(string(stdout)), nil
}

func TestJavaScriptClient(t *testing.T) {
	t.Run("client should be stable", func(t *testing.T) {
		testClientStability(t, javaScriptClient{})
	})

	t.Run("write client", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService2{})
		rpc.Register(&TestService1{})

		filePath := fmt.Sprintf("run-%d.js", rand.Int())
		err := rpc.WriteJavaScriptClient(filePath)
		t.Cleanup(func() {
			os.Remove(filePath)
		})

		assertNoError(t, err)
	})
}

func TestTypeScriptClient(t *testing.T) {
	t.Run("client should be stable", func(t *testing.T) {
		testClientStability(t, typeScriptClient{})
	})

	t.Run("write client", func(t *testing.T) {
		rpc := newTestServer()

		rpc.Register(&TestService2{})
		rpc.Register(&TestService1{})

		filePath := fmt.Sprintf("run-%d.ts", rand.Int())
		err := rpc.WriteTypeScriptClient(filePath)
		t.Cleanup(func() {
			os.Remove(filePath)
		})

		assertNoError(t, err)
	})
}

func TestGeneratedJavaScriptClient(t *testing.T) {
	if !runClientTests {
		t.Skip()
	}

	testCases := []struct {
		desc          string
		services      []any
		serverOptions []ServerOption
		code          string
		output        string
	}{
		{
			desc: "type check",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "TestService1", "Three", 0).then((res) => console.log(res))`,
			output: "3",
		},
		{
			desc: "rpc call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `(new TestService1(URL)).three(0).then((res) => console.log(res))`,
			output: "3",
		},
		{
			desc: "unified rpc call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `(new RPC(URL)).testService1.three(0).then((res) => console.log(res))`,
			output: "3",
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "NotFound", "Test", 0).catch((e) => console.log(e.message))`,
			output: `could not find service "NotFound"`,
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "TestService1", "NotFound", 0).catch((e) => console.log(e.message))`,
			output: `could not find method "NotFound"`,
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "TestService1", "Error", "test").catch((e) => console.log(e.message))`,
			output: `test`,
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "TestService1", "Error", "test").catch((e) => console.log(e instanceof RPCError))`,
			output: `true`,
		},
		{
			desc: "version",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `console.log((new RPC(URL)).version)`,
			output: `945207096c94fdab57aab31ff408884c46daebe9`,
		},
		{
			desc: "version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "TestService1", "Three", 0, "wrong version", () => console.log("mismatch"))`,
			output: "mismatch",
		},
		{
			desc: "no version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, {}, "TestService1", "Three", 0, "945207096c94fdab57aab31ff408884c46daebe9", () => console.log("no mismatch"))`,
			output: "",
		},
		{
			desc: "unified rpc version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `const rpc = new RPC(URL); rpc.testService1.clientVersion = "wrong version"; rpc.onVersionMismatch = () => console.log("mismatch"); rpc.testService1.three(0);`,
			output: "mismatch",
		},
		{
			desc: "no unified rpc version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `const rpc = new RPC(URL); rpc.onVersionMismatch = () => console.log("no mismatch"); rpc.testService1.three(0);`,
			output: "",
		},
	}

	for _, tC := range testCases {
		tC := tC
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()

			filePath := fmt.Sprintf("run-%d.js", rand.Int())

			rpc := newTestServer(tC.serverOptions...)

			for _, s := range tC.services {
				rpc.Register(s)
			}

			server := httptest.NewServer(rpc)

			t.Cleanup(func() {
				server.Close()
			})

			client := fmt.Sprintf("%s\n\nconst URL = %q;\n\n%s", rpc.JavaScriptClient(), server.URL, tC.code)

			err := os.WriteFile(filePath, []byte(client), 0600)

			assertNoError(t, err)

			t.Cleanup(func() {
				os.Remove(filePath)
			})

			output, err := execWithOutput("node", filePath)

			assertNoError(t, err)

			assertEqual(t, tC.output, output)
		})
	}
}

func TestGeneratedTypeScriptClient(t *testing.T) {
	if !runClientTests {
		t.Skip()
	}

	testCases := []struct {
		desc          string
		services      []any
		serverOptions []ServerOption
		code          string
		output        string
		headers       map[string]string
	}{
		{
			desc: "type check",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "TestService1", "Three", 0).then((res) => console.log(res))`,
			output: "3",
		},
		{
			desc: "rpc call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `(new TestService1(URL)).three(0).then((res) => console.log(res))`,
			output: "3",
		},
		{
			desc: "unified rpc call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `(new RPC(URL)).testService1.three(0).then((res) => console.log(res))`,
			output: "3",
		},
		{
			desc: "rpc call headers",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `(new TestService1(URL, {"test": "1234"})).three(0).then((res) => console.log(res))`,
			output: "3",
			headers: map[string]string{
				"test": "1234",
			},
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "NotFound", "Test", 0).catch((e) => console.log(e.message))`,
			output: `could not find service "NotFound"`,
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "TestService1", "NotFound", 0).catch((e) => console.log(e.message))`,
			output: `could not find method "NotFound"`,
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "TestService1", "Error", "test").catch((e) => console.log(e.message))`,
			output: `test`,
		},
		{
			desc: "call",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "TestService1", "Error", "test").catch((e) => console.log(e instanceof RPCError))`,
			output: `true`,
		},
		{
			desc: "version",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `console.log((new RPC(URL)).version)`,
			output: `945207096c94fdab57aab31ff408884c46daebe9`,
		},
		{
			desc: "version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "TestService1", "Three", 0, undefined, "wrong version", () => console.log("mismatch"))`,
			output: "mismatch",
		},
		{
			desc: "no version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `call(URL, "TestService1", "Three", 0, undefined, "945207096c94fdab57aab31ff408884c46daebe9", () => console.log("no mismatch"))`,
			output: "",
		},
		{
			desc: "unified rpc version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `const rpc = new RPC(URL); rpc.testService1.clientVersion = "wrong version"; rpc.onVersionMismatch = () => console.log("mismatch"); rpc.testService1.three(0);`,
			output: "mismatch",
		},
		{
			desc: "no unified rpc version mismatch",
			services: []any{
				&TestService1{},
				&TestService2{},
			},
			code:   `const rpc = new RPC(URL); rpc.onVersionMismatch = () => console.log("no mismatch"); rpc.testService1.three(0);`,
			output: "",
		},
	}

	for _, tC := range testCases {
		tC := tC
		t.Run(tC.desc, func(t *testing.T) {
			t.Parallel()

			filePath := fmt.Sprintf("run-%d.ts", rand.Int())

			rpc := newTestServer(tC.serverOptions...)

			for _, s := range tC.services {
				rpc.Register(s)
			}

			var header http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				header = r.Header
				rpc.ServeHTTP(w, r)
			}))

			t.Cleanup(func() {
				server.Close()
			})
			client := fmt.Sprintf(`
interface HeadersInit {
}

declare function fetch(url: string, input: object): Promise<Response>;
%s

const URL = %q;

%s
			`, rpc.TypeScriptClient(), server.URL, tC.code)

			err := os.WriteFile(filePath, []byte(client), 0600)

			assertNoError(t, err)

			t.Cleanup(func() {
				os.Remove(filePath)
			})

			_, err = execWithOutput("tsc", "--lib", "ES2015,dom", "--noEmit", "--strict", filePath)

			assertNoError(t, err)

			output, err := execWithOutput("tsx", filePath)

			assertNoError(t, err)

			assertEqual(t, tC.output, output)

			for name, value := range tC.headers {
				assertEqual(t, value, header.Get(name))
			}
		})
	}
}
