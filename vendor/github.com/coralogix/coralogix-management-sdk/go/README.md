# Coralogix Management SDK for Go

The Golang SDK for Coralogix uses (mostly) GRPC for interacting with the SaaS platform. Use `make` (install if required) to build and test.

## Important Makefile Targets

- `make build` - builds the binary in the default architecture 
- `make proto-clean` - clean the Go files created from protobuf
- `make proto-compile` - compile the Go files from protobuf
- `make test` - run examples/tests

# Troubleshooting

Here are some tips for getting everything to work:

```
panic: proto: extension number 5000 is already registered on message google.protobuf.MethodOptions
	previously from: "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/apm/common/v2"
	currently from:  "github.com/coralogix/coralogix-management-sdk/go/internal/coralogixapis/logs2metrics/v2"
See https://protobuf.dev/reference/go/faq#namespace-conflict
```
protoc treats conflicting extensions as an error unless this flag is specified (then it's a warning):

`-ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn"` 
