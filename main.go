package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/oiweiwei/midl-gen-go/codegen/golang"
	"github.com/oiweiwei/midl-gen-go/logx"
	"github.com/oiweiwei/midl-gen-go/midl"
	"github.com/oiweiwei/midl-gen-go/msdn/openspecs"
	"github.com/spf13/cobra"
)

// newSession builds the IDL session from -I flags followed by MSIDLPATH entries.
func newSession(includes []string, log *slog.Logger) *midl.Session {
	if env := os.Getenv("MSIDLPATH"); env != "" {
		includes = append(append([]string{}, includes...), strings.Split(env, ":")...)
	}
	return midl.NewSession(midl.WithInclude(includes...), midl.WithLogger(log))
}

// newLogger returns a stderr logger; debug level if verbose.
func newLogger(verbose bool) *slog.Logger {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
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
		emitTrace                 bool
		noFormat                  bool
		featuresSpec              string
	)

	cmd := &cobra.Command{
		Use:   "generate [flags] file.idl ...",
		Short: "Generate Go client stubs from IDL files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			log := newLogger(verbose)
			sess := newSession(includes, log)

			files, err := collectFiles(args)
			if err != nil {
				return fmt.Errorf("collect files: %w", err)
			}

			ctx := context.Background()

			features, err := golang.ParseFeatures(featuresSpec)
			if err != nil {
				return fmt.Errorf("--features: %w", err)
			}

			for _, fn := range files {
				p, err := golang.New(golang.Options{
					Session:              sess,
					Log:                  log,
					Features:             features,
					ImportsPath:          pkg,
					Format:               !noFormat,
					EmitTrace:            emitTrace,
					MSDNCache:            msdnOpenspecsCache,
					MSDNIndexerFile:      msdnOpenspecsIndexer,
					MSDNIndexerExtraFile: msdnOpenspecsIndexerExtra,
				})
				if err != nil {
					return fmt.Errorf("%s: %w", fn, err)
				}
				out, err := p.Gen(ctx, fn)
				if err != nil {
					return fmt.Errorf("%s: %w", fn, err)
				}
				if err := golang.WriteFiles(log, output, out); err != nil {
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
	cmd.Flags().StringVar(&featuresSpec, "features", "all",
		"what to generate besides the types and operations: a comma separated list of features, "+
			"\"all\", \"none\" and \"-name\" to leave one out (see the features command)")
	cmd.Flags().BoolVar(&emitTrace, "emit-trace", false, "add trace comments to the generated sources")
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
			file, err := newSession(includes, newLogger(false)).Load(args[0])
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

func newSymbolsCmd() *cobra.Command {
	var (
		includes []string
		kinds    []string
		match    string
		explain  bool
		asJSON   bool
	)

	cmd := &cobra.Command{
		Use:   "symbols [flags] file.idl ...",
		Short: "List the IDL names and the Go names they are given, and how",
		Long: `List the names of IDL files (constants, types, fields, union arms, enumeration elements,
interfaces, operations and parameters) with the Go name the generator gives each, the words
the name consists of and the naming rules that fired.

It uses the naming functions of the generator, so it shows what the generated code will have.
The OWNER column is the key of the "override" section of the gonames rules: to rename one
field of one struct, write

    override:
      OWNER:
        IDL: [Words, Of, The, Name]

Use --explain to see every step of the derivation of a name.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			var re *regexp.Regexp
			if match != "" {
				var err error
				if re, err = regexp.Compile(match); err != nil {
					return fmt.Errorf("--match: %w", err)
				}
			}

			want := map[string]bool{}
			for _, k := range kinds {
				want[k] = true
			}

			sess := newSession(includes, newLogger(false))

			files, err := collectFiles(args)
			if err != nil {
				return fmt.Errorf("collect files: %w", err)
			}

			var symbols []golang.Symbol

			for _, fn := range files {
				f, err := sess.Load(fn)
				if err != nil {
					return fmt.Errorf("%s: %w", fn, err)
				}
				p, err := golang.New(golang.Options{Session: sess})
				if err != nil {
					return err
				}
				for _, sym := range p.Symbols(context.Background(), f) {
					if len(want) > 0 && !want[sym.Kind] {
						continue
					}
					if re != nil && !re.MatchString(sym.Owner+"."+sym.IDL) && !re.MatchString(sym.Go) {
						continue
					}
					symbols = append(symbols, sym)
				}
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(symbols)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "KIND\tOWNER\tIDL\tGO\tWORDS\tRULES")
			for _, sym := range symbols {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", sym.Kind, sym.Owner, sym.IDL, sym.Go,
					strings.Join(sym.Trace.Words, " "), strings.Join(sym.Trace.Rules(), ","))
				if explain {
					if len(sym.Trace.Lexems) > 0 {
						fmt.Fprintf(w, "\t\t\t  lexems\t%s\t\n", strings.Join(sym.Trace.Lexems, " "))
					}
					for _, step := range sym.Trace.Steps {
						fmt.Fprintf(w, "\t\t\t  %s\t%s\t\n", step.Rule, step.Detail)
					}
				}
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringArrayVarP(&includes, "include", "I", nil,
		"IDL search path entry; repeatable. Format: path or base=path (base is Go module path)")
	cmd.Flags().StringSliceVar(&kinds, "kind", nil,
		"only these kinds: const, type, field, arm, element, interface, operation, param")
	cmd.Flags().StringVar(&match, "match", "", "only names whose OWNER.IDL or Go name matches the regular expression")
	cmd.Flags().BoolVar(&explain, "explain", false, "show every step of the derivation of each name")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")

	return cmd
}

func newFeaturesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "features",
		Short: "List the features of the generator that can be switched on and off",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "FEATURE\tREQUIRES\tDESCRIPTION")
			for _, f := range golang.Features() {
				fmt.Fprintf(w, "%s\t%s\t%s\n", f.Name(), strings.Join(f.Requires(), ","), f.Doc())
			}
			w.Flush()
		},
	}
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

			// page output goes to stdout, so logs go to stderr.
			ctx := logx.With(context.Background(), newLogger(verbose))

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

	root.AddCommand(newGenerateCmd(), newDumpCmd(), newSymbolsCmd(), newFeaturesCmd(), newMSDNCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
