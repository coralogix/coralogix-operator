// Command opgen generates operator kinds from the API contract.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/generator"
	"github.com/coralogix/coralogix-operator/v2/tools/opgen/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}
	switch args[0] {
	case "version", "--version":
		fmt.Println(version.Current)
		return 0
	case "generate":
		return runGenerate(args[1:])
	case "check":
		return runCheck(args[1:])
	}
	printUsage()
	return 2
}

func runGenerate(args []string) int {
	set := flag.NewFlagSet("opgen generate", flag.ContinueOnError)
	if err := set.Parse(args); err != nil {
		return 2
	}
	if set.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "opgen: unexpected arguments: %v\n", set.Args())
		return 2
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "opgen: find the current directory: %s\n", err)
		return 1
	}
	if err := generator.Generate(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "opgen: %s\n", err)
		return 1
	}
	return 0
}

func runCheck(args []string) int {
	set := flag.NewFlagSet("opgen check", flag.ContinueOnError)
	resource := set.String("resource", "", "OpenAPI component name of the resource")
	kind := set.String("kind", "", "custom resource kind (default: the resource name)")
	openAPI := set.String("openapi", "", "local candidate OpenAPI file")
	e2e := set.String("e2e", "", "e2e.yaml file of the kind")
	if err := set.Parse(args); err != nil {
		return 2
	}
	if set.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "opgen: unexpected arguments: %v\n", set.Args())
		return 2
	}
	if err := generator.Check(generator.CheckOptions{Kind: *kind, Resource: *resource, OpenAPIPath: *openAPI, E2EPath: *e2e}); err != nil {
		fmt.Fprintf(os.Stderr, "opgen: %s\n", err)
		return 1
	}
	return 0
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: opgen generate")
	fmt.Fprintln(os.Stderr, "       opgen check --resource <component> --openapi <candidate_openapi_file> [--kind <kind>] [--e2e <file>]")
	fmt.Fprintln(os.Stderr, "       opgen version")
}
