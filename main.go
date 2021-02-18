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

type method struct {
	Name          string
	NameLowerCase string
	Request       string
	Response      string
	HTTPMethod    string
	HTTPurl       string
}

// service contains all the necessary information to generate the files
type serivce struct {
	ImportPath          string
	Package             string
	ServiceName         string
	ServiceNameLoweCase string
	Methods             []method
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
		panic(fmt.Errorf("Failed to parse proto file %w", err))
	}

	for f, t := range generatedFiles {
		if err := createFileFromTemplate(svc, f, t); err != nil {
			panic(fmt.Errorf("Failed to create file %v: %w", f, err))
		}
	}

	// create the folder for go_out and grpc_out
	if err := os.Mkdir("./pkg/pb", 0775); !errors.Is(err, os.ErrExist) {
		panic(err)
	}
}

func parseProto() (svc serivce, err error) {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return svc, fmt.Errorf("ioutil.ReadAll(): %w", err)
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
	svc.ServiceNameLoweCase = firstLetterToLowerCase(svc.ServiceName)

	for _, m := range f.Services[0].Methods {
		svcMethod := method{}

		svcMethod.Name = strings.Title(m.GoName)
		svcMethod.NameLowerCase = firstLetterToLowerCase(svcMethod.Name)
		svcMethod.Request = m.Input.GoIdent.GoName
		svcMethod.Response = m.Output.GoIdent.GoName

		options, ok := m.Desc.Options().(*descriptorpb.MethodOptions)
		if !ok {
			return svc, fmt.Errorf("MethodDescriptor.Options() are not of type google.protobuf.descriptor")
		}

		httpRule, ok := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)
		if !ok {
			return svc, fmt.Errorf("proto.GetExtension(): method options are not of type google.api.httpRule")
		}

		switch httpRule.Pattern.(type) {
		case *annotations.HttpRule_Get:
			svcMethod.HTTPMethod = "GET"
			svcMethod.HTTPurl = httpRule.GetGet()
		case *annotations.HttpRule_Post:
			svcMethod.HTTPMethod = "POST"
			svcMethod.HTTPurl = httpRule.GetPost()
		case *annotations.HttpRule_Put:
			svcMethod.HTTPMethod = "PUT"
			svcMethod.HTTPurl = httpRule.GetPut()
		case *annotations.HttpRule_Patch:
			svcMethod.HTTPMethod = "PATCH"
			svcMethod.HTTPurl = httpRule.GetPatch()
		default:
			return svc, errors.New("HTTP method must be of type GET | POST | PUT | PATCH")
		}

		svc.Methods = append(svc.Methods, svcMethod)
	}

	return
}

func createFileFromTemplate(svc serivce, filePath, templatePath string) (err error) {
	filePath = strings.Replace(filePath, "{svc}", svc.ServiceNameLoweCase, -1)

	if _, err = os.Stat(filePath); os.IsExist(err) {
		// TODO promt user that file already exist and it will be truncated
	}

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
		return fmt.Errorf("Unable to read template file: %w", err)
	}

	t := template.New(templatePath)
	t, err = t.Parse(string(b))
	if err != nil {
		return fmt.Errorf("Unable to parse template file: %w", err)
	}

	if err = t.Execute(f, svc); err != nil {
		return fmt.Errorf("Error executing template %v:%w", t.Name(), err)
	}

	return
}

func firstLetterToLowerCase(s string) string {
	l := byte(unicode.ToLower(rune(s[0])))

	return string(append([]byte{l}, []byte(s)[1:]...))
}
