package parser

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

type Proto struct {
	Package      string
	GoImportPath string
	Services     []Service
}

type Service struct {
	Name      string
	NameTitle string
	Methods   []Method
}

type Method struct {
	Name      string
	NameTitle string
	Request   Message
	Response  Message
	Http      *Http
}

type Message struct {
	Name      string
	NameTitle string
	// Vars      map[string]string
}

type Http struct {
	Method    string
	URL       string
	Vars      []string
	VarsTitle []string
}

func Parse(r io.Reader) (p Proto, err error) {
	input, err := io.ReadAll(r)
	if err != nil {
		return p, fmt.Errorf("failed to read input: %w", err)
	}

	var req pluginpb.CodeGeneratorRequest
	if err = proto.Unmarshal(input, &req); err != nil {
		return p, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	opts := protogen.Options{}
	plugin, err := opts.New(&req)
	if err != nil {
		return p, fmt.Errorf("failed to create protogen.Plugin: %w", err)
	}

	// Files appear in topological order, so each file appears before any
	// file that imports it.
	f := plugin.Files[len(plugin.Files)-1]

	p.Package = string(f.GoPackageName)
	p.GoImportPath = filepath.Dir(string(f.GoImportPath))

	for i := range f.Services {
		methods := []Method{}
		for _, protoMethod := range f.Services[i].Methods {
			m, err := parseMethod(protoMethod)
			if err != nil {
				return p, fmt.Errorf("failed to parse method '%v' of service %v: %w", m.Name, f.Services[i].GoName, err)
			}
			methods = append(methods, m)
		}

		p.Services = append(p.Services, Service{
			Name:      firstLetterToLowerCase(f.Services[i].GoName),
			NameTitle: strings.Title(f.Services[i].GoName),
			Methods:   methods,
		})
	}

	return
}

func parseMethod(protoMethod *protogen.Method) (m Method, err error) {
	m.Name = firstLetterToLowerCase(protoMethod.GoName)
	m.NameTitle = strings.Title(protoMethod.GoName)

	// TODO parse message and its variables
	m.Request.Name = firstLetterToLowerCase(protoMethod.Input.GoIdent.GoName)
	m.Request.NameTitle = strings.Title(protoMethod.Input.GoIdent.GoName)
	m.Response.Name = firstLetterToLowerCase(protoMethod.Output.GoIdent.GoName)
	m.Response.NameTitle = strings.Title(protoMethod.Output.GoIdent.GoName)

	h, err := parseHTTP(protoMethod)
	if err != nil {
		return
	}
	m.Http = h

	return
}

func parseHTTP(protoMethod *protogen.Method) (h *Http, err error) {
	fmt.Fprintf(os.Stderr, "|%v| %T\n", protoMethod.Desc.Options(), protoMethod.Desc.Options())
	options, ok := protoMethod.Desc.Options().(*descriptorpb.MethodOptions)
	if !ok {
		return nil, fmt.Errorf("options are not of type google.protobuf.MethodOptions")
	}
	fmt.Fprintf(os.Stderr, "|%v| %T\n", options, options)

	if options != (*descriptorpb.MethodOptions)(nil) {
		if !proto.HasExtension(options, annotations.E_Http) {
			return nil, fmt.Errorf("options are not of type google.api.httpRule. (if there are no options, the '{}' must be removed and replaced with ';')")
		}

		httpRule := proto.GetExtension(options, annotations.E_Http).(*annotations.HttpRule)

		h = &Http{}

		switch httpRule.Pattern.(type) {
		case *annotations.HttpRule_Get:
			h.Method = http.MethodGet
			h.URL = httpRule.GetGet()
		case *annotations.HttpRule_Post:
			h.Method = http.MethodPost
			h.URL = httpRule.GetPost()
		case *annotations.HttpRule_Put:
			h.Method = http.MethodPut
			h.URL = httpRule.GetPut()
		case *annotations.HttpRule_Delete:
			h.Method = http.MethodDelete
			h.URL = httpRule.GetDelete()
		case *annotations.HttpRule_Patch:
			h.Method = http.MethodPatch
			h.URL = httpRule.GetPatch()
		default:
			return nil, errors.New("HTTP method must be of type GET | POST | PUT | DELETE | PATCH")
		}

		h.Vars = parseHTTPVariables(h.URL)
		if h.Vars != nil {
			for _, s := range h.VarsTitle {
				h.VarsTitle = append(h.VarsTitle, strings.Title(s))
			}
		}

		// TODO implement parseHTTPQueries()
	}

	return
}

func parseHTTPVariables(url string) []string {
	// TODO remove queries from beeing parsed
	f := func(c rune) bool {
		return c == '/'
	}
	fields := strings.FieldsFunc(url, f)

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

func firstLetterToLowerCase(s string) string {
	l := byte(unicode.ToLower(rune(s[0])))

	return string(append([]byte{l}, []byte(s)[1:]...))
}
