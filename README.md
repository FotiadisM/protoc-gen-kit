# protoc-gen-kit

wow such plugin o.O

1. [About](#about)
2. [Installation](#installation)
3. [Usage](#usage)
4. [Customization](#customization)

## About

Protoc-gen-kit is a highly customizable, 200 lines of code, [Protocol buffer](https://developers.google.com/protocol-buffers) compiler plugin that helps you build microservices faster. Protoc-gen-kit uses the [go-kit](https://github.com/go-kit/kit) library that strives to abide the Clean Architecture or the Hexagonal Architecture. Go-kit microservices are modeled like an onion. The innermost service domain is where everything is based on your specific service definition, and where all of the business logic is implemented. The middle endpoint domain is where each method of your service is abstracted to the generic [endpoint.Endpoint](https://pkg.go.dev/github.com/go-kit/kit/endpoint#Endpoint), and where safety and antifragile logic is implemented. Finally, the outermost transport domain is where endpoints are bound to concrete transports like HTTP or gRPC. Protoc-go-kit is inspired by [metaverse/truss](https://github.com/metaverse/truss).

By default protoc-gen-kit uses the default templates to create the serivce. But you can _very easily_ create your own templates that fit your style and needs.

## Installation

```
go get -u github.com/FotiadisM/protoc-gen-kit
```

## Usage

> You need to have protoc installed as well as the protoc-gen-go and protoc-gen-go-grpc plugin, for more information please visit the [gRPC website](https://grpc.io/docs/languages/go/quickstart/).

protoc-gen-kit supports both gRPC and HTPP.<br>
Clone this repo and navigate inside the example folder. There you file find called hello.proto.<br>
You define your service in protocol buffers like so:

```
# file hello.proto
syntax = "proto3";

package hello;

import "google/api/annotations.proto";

option go_package = "github.com/FotiadisM/protoc-gen-kit/example/hello";

service Hello {
  rpc Hello (HelloRequest) returns (HelloResponse) {
    option (google.api.http) = {
      get: "/hello"
    };
  }

  rpc HelloWith (HelloWithRequest) returns (HelloResponse) {
    option (google.api.http) = {
      get: "/hello/{name}"
    };
  }
}

message HelloRequest {}

message HelloWithRequest {
  string name = 1;
}

message HelloResponse {
  string Out = 1;
}

```

It's important that the option `go_package` is in the form of `<module path>/<service package>`.<br>
In the given example is the module path is `github.com/FotiadisM/protoc-gen-kit/example` and the serivce package is `hello`.

Then you generate the service by running:

```
protoc \
    -I $HOME/googleapis \
    --proto_path=. \
    --kit_out=. \
    --go_out=pkg/pb \
    --go_opt=paths=source_relative \
    --go-grpc_out=pkg/pb \
    --go-grpc_opt=paths=source_relative \
    hello.proto
```

Since protoc can be a bit confusing, let be explain

- `-I <path>` tells the compile where to search for the imported file, In my example I have placed them in `$HOME/googleapis`. You can download these files [here](https://github.com/googleapis/googleapis). The annotations.proto file is needed in order to describe HTTP routes for the Methods of the Service.
- `--proto_path=<path>` tells the compile where to look for the proto file. If you run this command inside the `example/` dir, the path should be `.`, this flag is required because protoc will otherwise look inside the `-I` directory.
- `--kit_out=<path>` When specifying `--kit_out` the protoc compiler will search for an executable binary with the name of `protoc_gen_kit` inside `$PATH`. The path variable is the path for the generated files.
- `--go_out` and `--go-grpc_out` invokes the the protoc-gen-go and protoc-gen-go-grpc plugins.
- `go_out=pahts=<path>` and `--go-grpc_opt=paths=<path>` adds some necessary flags for the go and grpc plugins.
- `hello.proto` Finaly the path to the proto file.

Your directory will now look like this:

```
examples/
│
├── cmd
│   └── hello
│       └── main.go
├── hello.proto
└── pkg
    ├── middleware
    │   ├── middleware.go
    │   └── wrap.go
    ├── pb
    │   ├── hello_grpc.pb.go
    │   └── hello.pb.go
    ├── server
    │   └── server.go
    └── svc
        ├── endpoints.go
        ├── service.go
        └── transport
            ├── grpc.go
            └── http.go
```

- `service.go` is where you will implement your business logic.
- `endpoints.go` containes all the endpoints and some helper functions.
- `/transport`
  - `http.go` containes the Request and Response structs and implements the HTTP routes.
  - `grpc.go` same as http.go but for gRPC
- `middleware.go` containes some predefined middleware for logging and metrics.
- `wrap.go` wrap containes functions that wrap your service and endpoints with middleware in a very easy way.

## Customization

Protoc-gen-kit will look for templates inside the template folder. In order to create a file, you have to include it inside the `generatedFiles` variable alongside with the template to use. You can easily make changes in the templates and change the generated code, or create your own, drop them inside the `templates/` directory
and then include them in `generatedFiles`. Everything will makes sence once you take a look a the code.
For reference, the default service is specified like so:

```go
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
```

Compile and you are set.
