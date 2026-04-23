package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/gen"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/spf13/cobra"
)

func applyIncludes(includes []string) {
	if len(includes) == 0 {
		return
	}
	existing := midl.GetPathVar()
	parts := includes
	if existing != "" {
		parts = append(parts, strings.Split(existing, ":")...)
	}
	midl.SetPathVar(strings.Join(parts, ":"))
}

func newGenerateCmd() *cobra.Command {
	var (
		includes []string
		output   string
		pkg      string
		docCache string
		verbose  bool
		noFormat bool
	)

	cmd := &cobra.Command{
		Use:   "generate [flags] file.idl ...",
		Short: "Generate Go client stubs from IDL files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyIncludes(includes)
			midl.RPCErrorVerbose = true
			midl.Setup()

			for _, fn := range args {
				p := &gen.Generator{
					ImportsPath: pkg,
					Format:      !noFormat,
					Trace:       verbose,
					Dir:         output,
					Cache:       docCache,
				}
				if err := p.Gen(context.Background(), fn); err != nil {
					return fmt.Errorf("%s: %w", fn, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&includes, "include", "I", nil,
		"IDL search path entry; repeatable. Format: path or base=path (base is Go module path)")
	cmd.Flags().StringVarP(&output, "output", "o", "msrpc/", "output directory root")
	cmd.Flags().StringVar(&pkg, "pkg", "github.com/oiweiwei/go-msrpc/msrpc", "Go import path base for generated packages")
	cmd.Flags().StringVar(&docCache, "doc-cache", ".cache/doc/", "cache directory for MSDN documentation")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "enable verbose/trace output")
	cmd.Flags().BoolVar(&noFormat, "no-format", false, "skip gofmt on generated files")

	return cmd
}

func newDumpCmd() *cobra.Command {
	var includes []string

	cmd := &cobra.Command{
		Use:   "dump [flags] file.idl",
		Short: "Dump parsed IDL as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyIncludes(includes)
			midl.Setup()

			file, err := midl.NewFile(args[0], "").Load()
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(file, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, string(b))
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&includes, "include", "I", nil,
		"IDL search path entry; repeatable. Format: path or base=path (base is Go module path)")

	return cmd
}

func main() {
	root := &cobra.Command{
		Use:           "midl-gen-go",
		Short:         "MIDL parser and Go client stub generator",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newGenerateCmd(), newDumpCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
