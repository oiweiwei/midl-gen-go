package midl

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Session owns everything that used to be process-wide state: the set of loaded
// files, the include path and the per-directory configuration cache. Two
// sessions never observe each other, so several IDL sets may be parsed in one
// process (or concurrently) without contaminating each other.
//
// A Session is safe for concurrent use.
type Session struct {
	log      *slog.Logger
	include  []*Path
	noBuiltn bool

	mu      sync.Mutex
	files   map[string]*File
	configs map[string]*Config
}

// SessionOption configures a Session.
type SessionOption func(*Session)

// WithInclude appends IDL search path entries. An entry is either "dir" or
// "gomodule=dir" (the Go module path the generated packages are rooted at).
func WithInclude(entries ...string) SessionOption {
	return func(s *Session) {
		for _, e := range entries {
			if e != "" {
				s.include = append(s.include, ParsePathEntry(e))
			}
		}
	}
}

// WithLogger sets the session logger. The default discards everything.
func WithLogger(l *slog.Logger) SessionOption {
	return func(s *Session) {
		if l != nil {
			s.log = l
		}
	}
}

// WithoutBuiltins skips preloading dcetypes.idl (useful for parsing
// self-contained snippets in tests).
func WithoutBuiltins() SessionOption {
	return func(s *Session) { s.noBuiltn = true }
}

// NewSession creates a session. Unless WithoutBuiltins is given, dcetypes.idl
// is loaded from the include path; failing to find it is logged as a warning.
func NewSession(opts ...SessionOption) *Session {
	s := &Session{
		log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		files:   make(map[string]*File),
		configs: make(map[string]*Config),
	}
	for _, opt := range opts {
		opt(s)
	}
	if !s.noBuiltn {
		if _, err := s.Load("dcetypes.idl"); err != nil {
			s.log.Warn("unable to load dcetypes.idl", "err", err)
		}
	}
	return s
}

// Logger returns the session logger.
func (s *Session) Logger() *slog.Logger { return s.log }

// Load finds, parses and registers the file; repeated loads of the same
// resolved path return the cached *File.
func (s *Session) Load(p string) (*File, error) {

	f := NewFile(p, "")

	path, err := s.FindFile(f.Path)
	if err != nil {
		return f, fmt.Errorf("midl: find file: %s: %w", f.Path, err)
	}

	if rf, ok := s.Lookup(path.File); ok {
		return rf, nil
	}

	b, err := os.ReadFile(path.File)
	if err != nil {
		return f, fmt.Errorf("midl: load file: %s: %w", path.File, err)
	}

	rf, err := s.Parse(string(b))
	if err != nil {
		return f, fmt.Errorf("midl: parse file: %w", err)
	}

	rf.FullPath = path.File
	rf.GoPkg = filepath.Join(path.Base, f.GoPkg)
	rf.Path = filepath.Join(path.Base, filepath.Base(f.Path))
	rf.GoPkgBase = path.GoBase
	rf.MIDLConfig = path.Config

	// process lib files.
	for _, lib := range rf.Libraries {
		dir, pkg := filepath.Dir(rf.FullPath), strings.ToLower(lib.Name)
		pkgFile := pkg + ".idl"
		ff := lib.Body
		ff.FullPath = filepath.Join(dir, rf.GoPkg, pkgFile)
		ff.GoPkg = filepath.Join(rf.GoPkg, pkg)
		ff.GoPkgBase = rf.GoPkgBase
		ff.Path = filepath.Join(rf.GoPkg, pkgFile)
		ff.MIDLConfig = rf.MIDLConfig
		ff.sess = s

		s.Register(ff.FullPath, ff)
	}

	s.Register(path.File, rf)

	return rf, nil
}

// Lookup returns the file registered under the resolved path.
func (s *Session) Lookup(p string) (*File, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.files[p]
	return f, ok
}

// Register records the file under the resolved path.
func (s *Session) Register(p string, f *File) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[p] = f
}

// Files returns all loaded files sorted by their resolved path.
func (s *Session) Files() []*File {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.files))
	for k := range s.files {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	ret := make([]*File, len(keys))
	for i := range keys {
		ret[i] = s.files[keys[i]]
	}

	return ret
}

// LookupType looks for an exported type by name in all loaded files.
func (s *Session) LookupType(n string) *Type {
	for _, f := range s.Files() {
		t, ok := f.exportSyms[n]
		if !ok || t.Type == nil {
			continue
		}
		return t.Type
	}
	return nil
}

// Path is an include path entry resolved for a particular file.
type Path struct {
	Dir    string
	Base   string
	File   string
	GoBase string
	Config *Config
}

// ParsePathEntry parses "dir" or "gomodule=dir".
func ParsePathEntry(e string) *Path {
	if k, v, ok := strings.Cut(e, "="); ok {
		return &Path{Dir: v, GoBase: k}
	}
	return &Path{Dir: e}
}

// SearchPath returns the directories searched for IDL files: the explicit
// includes first, then the working directory.
func (s *Session) SearchPath() []*Path {
	paths := make([]*Path, 0, len(s.include)+1)
	for _, p := range s.include {
		cp := *p
		paths = append(paths, &cp)
	}
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, &Path{Dir: wd})
	}
	return paths
}

// FindFile resolves the file name against the include path. Within each
// include entry the lookup order is: the qualified name ("sub/x.idl" as
// given), the bare file name in the entry itself, then in each first-level sub
// directory. Hence a file system path like "idl/x.idl" resolves to the include
// entry that contains x.idl. The "ms-" and "mc-" prefixes are optional.
func (s *Session) FindFile(name string) (*Path, error) {

	reldir, base := "", name
	if idx := strings.LastIndex(name, "/"); idx > 0 {
		reldir, base = name[:idx], name[idx+1:]
	}

	candidates := uniq(strings.TrimPrefix(base, "ms-"), strings.TrimPrefix(base, "mc-"), base)

	for _, p := range s.SearchPath() {

		reldirs := []string{""}
		if reldir != "" {
			reldirs = []string{reldir, ""}
		}

		subs, err := filepath.Glob(filepath.Join(p.Dir, "*"))
		if err != nil {
			return nil, err
		}

		for _, sub := range subs {
			if info, _ := os.Stat(sub); info != nil && info.IsDir() {
				reldirs = append(reldirs, filepath.Base(sub))
			}
		}

		for _, rd := range reldirs {
			dir := filepath.Join(p.Dir, rd)
			for _, file := range candidates {
				path := filepath.Join(dir, file)
				if _, err := os.Stat(path); err != nil {
					continue
				}

				cc, err := s.ConfigFor(dir, p.Dir)
				if err != nil {
					return nil, fmt.Errorf("cannot load config for file %q: %w", path, err)
				}

				if p.GoBase == "" && cc != nil {
					p.GoBase = cc.Go.Pkg
				}

				return &Path{
					File:   path,
					Base:   rd,
					Dir:    p.Dir,
					GoBase: p.GoBase,
					Config: cc.For(path),
				}, nil
			}
		}
	}

	dirs := make([]string, 0, len(s.include))
	for _, p := range s.include {
		dirs = append(dirs, p.Dir)
	}

	return nil, fmt.Errorf("cannot find file %q: path: %s", name, strings.Join(dirs, ":"))
}

// ConfigFor returns the first configuration found in the given directories.
// The first error encountered is reported only if no configuration is found.
func (s *Session) ConfigFor(dirs ...string) (*Config, error) {
	var first error
	for _, dir := range dirs {
		cc, err := s.LoadConfig(dir)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		if cc != nil {
			return cc, nil
		}
	}
	return nil, first
}

func uniq(ss ...string) []string {
	seen := make(map[string]bool, len(ss))
	out := ss[:0:0]
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
