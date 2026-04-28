package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oiweiwei/midl-gen-go/codegen/gen"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
	"github.com/rs/zerolog"
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

func collectFiles(paths []string) ([]string, error) {

	var files []string
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				files = append(files, path)
				continue
			}
			return nil, err
		}

		if stat.IsDir() {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".idl") {
					files = append(files, p)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			files = append(files, path)
		}
	}

	return files, nil
}

func newGenerateCmd() *cobra.Command {
	var (
		includes                  []string
		output                    string
		pkg                       string
		msdnOpenspecsCache        string
		msdnOpenspecsIndexer      string
		msdnOpenspecsIndexerExtra string
		verbose                   bool
		noFormat                  bool
	)

	cmd := &cobra.Command{
		Use:   "generate [flags] file.idl ...",
		Short: "Generate Go client stubs from IDL files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyIncludes(includes)
			midl.RPCErrorVerbose = true
			midl.Setup()

			files, err := collectFiles(args)
			if err != nil {
				return fmt.Errorf("collect files: %w", err)
			}

			for _, fn := range files {

				ctx := context.Background()

				if verbose {
					ctx = zerolog.New(os.Stdout).Level(zerolog.DebugLevel).WithContext(ctx)
				}

				p := &gen.Generator{
					ImportsPath:          pkg,
					Format:               !noFormat,
					Trace:                verbose,
					Dir:                  output,
					MSDNCache:            msdnOpenspecsCache,
					MSDNIndexerFile:      msdnOpenspecsIndexer,
					MSDNIndexerExtraFile: msdnOpenspecsIndexerExtra,
				}
				if err := p.Gen(ctx, fn); err != nil {
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
	cmd.Flags().StringVar(&msdnOpenspecsCache, "msdn-openspecs-cache-dir", "msdn/.cache/doc/", "cache directory for MSDN documentation")
	cmd.Flags().StringVar(&msdnOpenspecsIndexer, "msdn-openspecs-indexer-file", "", "indexer file for MSDN documentation")
	cmd.Flags().StringVar(&msdnOpenspecsIndexerExtra, "msdn-openspecs-indexer-extra-file", "", "extra indexer file for MSDN documentation; repeatable")
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

func newMSDNCmd() *cobra.Command {
	var (
		msdnOpenspecsCache        string
		msdnOpenspecsIndexer      string
		msdnOpenspecsIndexerExtra string
		list                      bool
		verbose                   bool
		outputFormat              string
	)

	cmd := &cobra.Command{
		Use:   "msdn [flags] index-name [object-name...]",
		Short: "Fetch and render an MSDN Open Specifications documentation page",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			protocolName, pageNames := args[0], args[1:]

			ctx := context.Background()
			if verbose {
				ctx = zerolog.New(os.Stdout).Level(zerolog.DebugLevel).WithContext(ctx)
			}

			indexer, err := openspecs.NewProtocolIndexerFromFile(msdnOpenspecsIndexer)
			if err != nil {
				return fmt.Errorf("msdn: openspecs: load indexer file: %w", err)
			}

			if msdnOpenspecsIndexerExtra != "" {
				if err := indexer.ReadExtraFromFile(msdnOpenspecsIndexerExtra); err != nil {
					return fmt.Errorf("msdn: openspecs: read extra indexer file: %w", err)
				}
			}

			msdn := &openspecs.MSDN{
				CacheFS: msdnOpenspecsCache,
				Indexer: indexer,
			}

			if err := msdn.Sync(ctx, protocolName); err != nil {
				return fmt.Errorf("msdn: openspecs: sync: %w", err)
			}

			if list || len(pageNames) == 0 {
				index, ok := msdn.Index(ctx, protocolName)
				if !ok {
					return fmt.Errorf("msdn: index not found for protocol: %s", protocolName)
				}
				index.Each(func(entry string, _ map[string]string) bool {
					if name, ok := openspecs.ExtractName(entry); ok {
						fmt.Println(name)
					}
					return true
				})
				return nil
			}

			page, ok := msdn.GetPage(ctx, pageNames...)
			if !ok {
				return fmt.Errorf("msdn: page not found: %s", strings.Join(pageNames, ", "))
			}

			if outputFormat == "json" {
				b, err := json.MarshalIndent(page, "", "  ")
				if err != nil {
					return fmt.Errorf("msdn: marshal page: %w", err)
				}
				fmt.Println(string(b))
				return nil
			}

			fmt.Print(page.Render())
			return nil
		},
	}

	cmd.Flags().StringVar(&msdnOpenspecsCache, "msdn-openspecs-cache-dir", "msdn/.cache/", "cache directory for MSDN documentation")
	cmd.Flags().StringVar(&msdnOpenspecsIndexer, "msdn-openspecs-indexer-file", "msdn/index.yaml", "indexer file for MSDN documentation")
	cmd.Flags().StringVar(&msdnOpenspecsIndexerExtra, "msdn-openspecs-indexer-extra-file", "msdn/extra.yaml", "extra indexer file for MSDN documentation")
	cmd.Flags().BoolVar(&list, "list", false, "list available object names in the protocol index")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "enable verbose/trace output")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format: json (default: text render)")

	return cmd
}

func main() {
	root := &cobra.Command{
		Use:           "midl-gen-go",
		Short:         "MIDL parser and Go client stub generator",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newGenerateCmd(), newDumpCmd(), newMSDNCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
