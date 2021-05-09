package main

import (
	"os"

	"github.com/FotiadisM/protoc-gen-kit/internal/generator"
	"github.com/FotiadisM/protoc-gen-kit/pkg/parser"
)

func main() {
	p, err := parser.Parse(os.Stdin)
	if err != nil {
		panic(err)
	}

	appName := p.Parameters["appName"]
	c := generator.Config{
		AppName:  appName,
		TemplDir: "./templates",
		Proto:    p,
	}
	err = generator.Generate(c)
	if err != nil {
		panic(err)
	}
}
