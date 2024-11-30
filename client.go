package turborpc

import (
	"os"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	_ "embed"

	"github.com/olahol/tsreflect"
)

//go:embed javascript.tmpl
var javascriptTemplateText string

//go:embed typescript.tmpl
var typescriptTemplateText string

type sourceClient struct {
	ContentType string
	SourceCode  string
}

// A clientGenerator interface is used by a server to generate source code for
// a client that can access its services.
type clientGenerator interface {
	// GenerateClient returns the source code of the client.
	GenerateClient(serverMetadata) sourceClient
}

type javaScriptClient struct {
}

func newJavaScriptClient() (jc javaScriptClient) {
	return jc
}

func (c javaScriptClient) GenerateClient(metadata serverMetadata) sourceClient {
	return sourceClient{
		ContentType: "text/javascript",
		SourceCode:  generateClientFromTemplate(javascriptTemplateText, metadata),
	}
}

type typeScriptClient struct {
}

func newTypeScriptClient() (tc typeScriptClient) {
	return tc
}

func (c typeScriptClient) GenerateClient(metadata serverMetadata) sourceClient {
	return sourceClient{
		ContentType: "application/typescript",
		SourceCode:  generateClientFromTemplate(typescriptTemplateText, metadata),
	}
}

// clientSourceCode generates a client for the server.
func (rpc *Server) clientSourceCode(client clientGenerator) string {
	g := client.GenerateClient(rpc.metadata())
	return g.SourceCode
}

// writeClientSourceCode write a client for the server to a file.
func (rpc *Server) writeClientSourceCode(client clientGenerator, filePath string) error {
	return os.WriteFile(filePath, []byte(rpc.clientSourceCode(client)), 0600)
}

// TypeScriptClient returns source code for a TypeScript client.
// The TypeScript client is implemented as an ES6 module where each service is
// a class. The method names are converted from pascal case to camel case
// i.e "MyMethod" becomes "myMethod". There is also a class containing all
// services with the name RPC. The classes are instantiated with the
// URL endpoint of the rpc http server and optional headers that are passed to
// the server.
func (rpc *Server) TypeScriptClient() string {
	return rpc.clientSourceCode(newTypeScriptClient())
}

// WriteTypeScriptClient writes a TypeScript client to a file.
func (rpc *Server) WriteTypeScriptClient(filePath string) error {
	return rpc.writeClientSourceCode(newTypeScriptClient(), filePath)
}

// MustWriteTypeScriptClient generates a TypeScript client and writes it to the specified file path.
// If an error occurs during the generation or writing process, it will panic.
func (rpc *Server) MustWriteTypeScriptClient(filePath string) {
	if err := rpc.WriteTypeScriptClient(filePath); err != nil {
		panic(err)
	}
}

// JavaScriptClient returns source code for a JavaScript client.
func (rpc *Server) JavaScriptClient() string {
	return rpc.clientSourceCode(newJavaScriptClient())
}

// WriteJavaScriptClient writes a JavaScript client to a file.
func (rpc *Server) WriteJavaScriptClient(filePath string) error {
	return rpc.writeClientSourceCode(newJavaScriptClient(), filePath)
}

// MustWriteJavaScriptClient generates a JavaScript client and writes it to the specified file path.
// If an error occurs during the generation or writing process, it will panic.
func (rpc *Server) MustWriteJavaScriptClient(filePath string) {
	if err := rpc.WriteJavaScriptClient(filePath); err != nil {
		panic(err)
	}
}

func isVoid(typ reflect.Type) bool {
	return typ == nil
}

func camelCase(s string) string {
	rs := []rune(s)
	rs[0] = unicode.ToLower(rs[0])
	return string(rs)
}

func generateClientFromTemplate(templateText string, metadata serverMetadata) string {
	g := tsreflect.New(tsreflect.WithFlatten(), tsreflect.WithNamer(tsreflect.PackageNamer))

	for _, typ := range metadata.types() {
		g.Add(typ)
	}

	funcs := template.FuncMap{
		"camelCase": camelCase,
		"typeOf":    g.TypeOf,
		"isVoid":    isVoid,
	}

	tmpl := template.Must(template.New("").Funcs(funcs).Parse(templateText))

	var sb strings.Builder
	_ = tmpl.Execute(&sb, map[string]any{
		"DatePrefix":        datePrefix,
		"Metadata":          metadata,
		"SymbolsJSDoc":      g.DeclarationsJSDoc(),
		"SymbolsTypeScript": g.DeclarationsTypeScript(),
	})

	return sb.String()
}
