package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/FotiadisM/protoc-gen-kit/pkg/parser"
	"gopkg.in/yaml.v2"
)

const (
	generatorFileNane string = "generator.yaml"
)

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
	g, err := parseGeneratorFile(filepath.Join(c.TemplDir, generatorFileNane))
	if err != nil {
		return
	}

	for _, gFile := range g.Files {
		genPath := strings.ReplaceAll(gFile.GenPath, "{AppName}", c.AppName)
		f, err := createFile(genPath)
		if err != nil {
			return err
		}

		tmplPath := filepath.Join(c.TemplDir, gFile.TmplPath)
		err = parseTemplate(f, tmplPath, c.Proto)
		if err != nil {
			return err
		}

		f.Close()
	}

	return
}

func parseGeneratorFile(genFilePath string) (g generatorFile, err error) {
	f, err := os.Open(genFilePath)
	if err != nil {
		return g, fmt.Errorf("failed to open %v: %w", genFilePath, err)
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&g)
	if err != nil {
		return g, fmt.Errorf("failed to decode %v: %w", genFilePath, err)
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

func parseTemplate(f *os.File, tmplPath string, proto parser.Proto) error {
	t, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to parse template %v: %w", tmplPath, err)
	}

	err = t.Execute(f, proto)
	if err != nil {
		return fmt.Errorf("failed to execute template %v: %w", tmplPath, err)
	}

	return nil
}
