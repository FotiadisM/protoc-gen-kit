package generator

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/FotiadisM/protoc-gen-kit/pkg/parser"
	"gopkg.in/yaml.v2"
)

const (
	generatorFileNane       string = "generator.yaml"
	defaultTemplatesDirName string = "templates"
)

//go:embed templates
var defaultTemplates embed.FS

type Config struct {
	TemplDir string
	Proto    parser.Proto
	AppName  string
}

type generatorFile struct {
	Files []struct {
		TmplPath string `yaml:"templatePath"`
		GenPath  string `yaml:"generatorPath"`
	} `yaml:"files"`
}

func Generate(c Config) (err error) {
	var g generatorFile

	// opening and parsing generator file
	var f fs.File
	if c.TemplDir == "" { // use default template
		f, err = defaultTemplates.Open(filepath.Join(defaultTemplatesDirName, generatorFileNane))
		if err != nil {
			return fmt.Errorf("failed to open %v of the default template: %w", generatorFileNane, err)
		}
	} else {
		f, err = os.Open(filepath.Join(c.TemplDir, generatorFileNane))
		if err != nil {
			return fmt.Errorf("failed to open %v: %w", filepath.Join(c.TemplDir, generatorFileNane), err)
		}
	}
	g, err = parseGeneratorFile(f)
	if err != nil {
		return err
	}
	f.Close()

	// creating files and executing templates
	for _, gFile := range g.Files {
		if strings.Contains(gFile.GenPath, "{AppName}") {
			if c.AppName == "" {
				return fmt.Errorf("generator path contains '{AppName}' but no app name was specified, use the flag -app string")
			}
			gFile.GenPath = strings.ReplaceAll(gFile.GenPath, "{AppName}", c.AppName)
		}
		f, err := createFile(gFile.GenPath)
		if err != nil {
			return err
		}

		var templateString []byte
		if c.TemplDir == "" { //use default template
			templateString, err = defaultTemplates.ReadFile(filepath.Join(defaultTemplatesDirName, gFile.TmplPath))
			if err != nil {
				return fmt.Errorf("failed to read file %v of the defualt template: %w", gFile.TmplPath, err)
			}
		} else {
			templateString, err = os.ReadFile(filepath.Join(c.TemplDir, gFile.TmplPath))
			if err != nil {
				return fmt.Errorf("failed to read file %v: %w", gFile.TmplPath, err)
			}
		}
		err = parseTemplate(f, string(templateString), c.Proto)
		if err != nil {
			return fmt.Errorf("template: %v: %w", gFile.TmplPath, err)
		}

		f.Close()
	}

	return
}

func parseGeneratorFile(f io.Reader) (g generatorFile, err error) {
	err = yaml.NewDecoder(f).Decode(&g)
	if err != nil {
		return g, fmt.Errorf("failed to decode %v: %w", generatorFileNane, err)
	}

	return
}

func createFile(path string) (f *os.File, err error) {
	if err = os.MkdirAll(filepath.Dir(path), 0775); err != nil {
		return nil, fmt.Errorf("failed to create directory/directories %v: %w", filepath.Dir(path), err)
	}

	f, err = os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %v: %w", path, err)
	}

	return
}

func parseTemplate(f *os.File, templateString string, proto parser.Proto) error {
	t, err := template.New("genericName").Parse(templateString)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	err = t.Execute(f, proto)
	if err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}

	return nil
}
