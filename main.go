package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// templateDir is the direcotry that createFile will look
// for the given template
//go:embed templates
var templateDir embed.FS

type http struct {
	Method        string
	URI           string
	Vars          []string
	VarsLowerCase []string
}

type method struct {
	Name          string
	NameLowerCase string
	Request       string
	Response      string
	HTTP          *http
}

// service contains all the necessary information to generate the files
type serivce struct {
	ImportPath           string
	Package              string
	ServiceName          string
	ServiceNameLowerCase string
	Methods              []method
}

// generatedFiles containes all the files that are going to be created
// The Key containes the path to the file to be created while
// the Value contains the path to the template to be used
// The template must be contained inside the templateDir folder
// {svc} will be replaced by the service name
var generatedFiles = map[string]string{
	"./cmd/{svc}/main.go":            "templates/cmd/main.tmpl",
	"./pkg/middleware/middleware.go": "templates/middleware/middleware.tmpl",
	"./pkg/middleware/wrap.go":       "templates/middleware/wrap.tmpl",
	"./pkg/server/server.go":         "templates/server/server.tmpl",
	"./pkg/svc/service.go":           "templates/svc/service.tmpl",
	"./pkg/svc/endpoints.go":         "templates/svc/endpoints.tmpl",
	"./pkg/svc/transport/grpc.go":    "templates/svc/transport/grpc.tmpl",
	"./pkg/svc/transport/http.go":    "templates/svc/transport/http.tmpl",
}

func main() {
	svc, err := parseProto()
	if err != nil {
		panic(fmt.Errorf("failed to parse proto file %w", err))
	}

	for f, t := range generatedFiles {
		if err := createFileFromTemplate(svc, f, t); err != nil {
			panic(fmt.Errorf("failed to create file %v: %w", f, err))
		}
	}

	// create the folder for go_out and grpc_out
	if err := os.Mkdir("./pkg/pb", 0775); err != nil {
		if errors.Is(err, os.ErrExist) {
			panic(err)
		}
	}
}

func parseProto() (svc serivce, err error) {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return svc, fmt.Errorf("io.ReadAll(): %w", err)
	}

	req := pluginpb.CodeGeneratorRequest{}
	if err = proto.Unmarshal(input, &req); err != nil {
		return svc, fmt.Errorf("proto.Unmarshal(): %w", err)
	}

	opts := protogen.Options{}
	plugin, err := opts.New(&req)
	if err != nil {
		return svc, fmt.Errorf("protogen.Options.New(): %w", err)
	}

	// Files appear in topological order, so each file appears before any
	// file that imports it.
	f := plugin.Files[len(plugin.Files)-1]

	svc.Package = string(f.GoPackageName)
	svc.ImportPath = filepath.Dir(string(f.GoImportPath))
	svc.ServiceName = strings.Title(f.Services[0].GoName)
	svc.ServiceNameLowerCase = firstLetterToLowerCase(svc.ServiceName)

	for _, m := range f.Services[0].Methods {
		var svcMethod method

		svcMethod, err = parseMethod(m)
		if err != nil {
			return
		}

		svc.Methods = append(svc.Methods, svcMethod)
	}

	return
}

func parseMethod(m *protogen.Method) (svcMethod method, err error) {

	svcMethod.Name = strings.Title(m.GoName)
	svcMethod.NameLowerCase = firstLetterToLowerCase(svcMethod.Name)
	svcMethod.Request = m.Input.GoIdent.GoName
	svcMethod.Response = m.Output.GoIdent.GoName

	options, ok := m.Desc.Options().(*descriptorpb.MethodOptions)
	if !ok {
		return svcMethod, fmt.Errorf("method %v: options are not of type google.protobuf.MethodOptions", m.GoName)
	}

	if options != (*descriptorpb.MethodOptions)(nil) {
		if !proto.HasExtension(options, annotations.E_Http) {
			return svcMethod, fmt.Errorf("method %v: options are not of type google.api.httpRule", m.GoName)
		}

		httpRule := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)

		ht := &http{}

		switch httpRule.Pattern.(type) {
		case *annotations.HttpRule_Get:
			ht.Method = "GET"
			ht.URI = httpRule.GetGet()
		case *annotations.HttpRule_Post:
			ht.Method = "POST"
			ht.URI = httpRule.GetPost()
		case *annotations.HttpRule_Put:
			ht.Method = "PUT"
			ht.URI = httpRule.GetPut()
		case *annotations.HttpRule_Patch:
			ht.Method = "PATCH"
			ht.URI = httpRule.GetPatch()
		default:
			return svcMethod, errors.New("HTTP method must be of type GET | POST | PUT | PATCH")
		}

		ht.VarsLowerCase = parseHTTPVariables(ht.URI)
		for _, s := range ht.VarsLowerCase {
			ht.Vars = append(ht.Vars, strings.Title(s))
		}

		svcMethod.HTTP = ht
	}

	return
}

func parseHTTPVariables(uri string) []string {
	f := func(c rune) bool {
		return c == '/'
	}
	fields := strings.FieldsFunc(uri, f)

	var vars []string
	for _, s := range fields {
		if s[0] == '{' {
			if i := strings.Index(s, ":"); i != -1 {
				vars = append(vars, s[1:i])
				continue
			}

			i := strings.Index(s, "}")
			vars = append(vars, s[1:i])
		}
	}

	return vars
}

func createFileFromTemplate(svc serivce, filePath, templatePath string) (err error) {
	filePath = strings.Replace(filePath, "{svc}", svc.ServiceNameLowerCase, -1)

	// TODO promt user that file already exist and it will be truncated
	// if _, err = os.Stat(filePath); os.IsExist(err) {
	// }

	if err = os.MkdirAll(filepath.Dir(filePath), 0775); err != nil {
		return fmt.Errorf("os.MkdirAll() %w", err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("os.Create(): %w", err)
	}
	defer f.Close()

	b, err := templateDir.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("unable to read template file: %w", err)
	}

	t := template.New(templatePath)
	t, err = t.Parse(string(b))
	if err != nil {
		return fmt.Errorf("unable to parse template file: %w", err)
	}

	if err = t.Execute(f, svc); err != nil {
		return fmt.Errorf("error executing template %v:%w", t.Name(), err)
	}

	return
}

func firstLetterToLowerCase(s string) string {
	l := byte(unicode.ToLower(rune(s[0])))

	return string(append([]byte{l}, []byte(s)[1:]...))
}
