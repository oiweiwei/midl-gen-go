package midl

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	go_names "github.com/oiweiwei/midl-gen-go/gonames"
)

var (
	fileStore   = make(map[string]*File)
	fileStoreMu = new(sync.Mutex)
)

// Load function loads the file content.
func (f *File) Load() (*File, error) {

	var path *Path
	var err error

	if path, err = FindFile(f.Path); err != nil {
		if err != nil {
			return f, fmt.Errorf("midl: find file: %s: %w", f.Path, err)
		}
	}

	if rf, ok := FileLoad(path.File); ok {
		return rf, nil
	}

	b, err := os.ReadFile(path.File)
	if err != nil {
		return f, fmt.Errorf("midl: load file: %s: %w", path.File, err)
	}
	rf, err := Parse(string(b))
	if err != nil {
		return f, fmt.Errorf("midl: parse file: %w", err)
	}

	rf.FullPath = path.File
	rf.GoPkg = filepath.Join(path.Base, f.GoPkg)
	rf.Path = filepath.Join(path.Base, filepath.Base(f.Path))
	rf.GoPkgBase = path.GoBase
	rf.MIDLConfig = path.Config

	*f = *rf

	FileStore(path.File, f)

	return f, nil
}

// FileLoad ...
func FileLoad(p string) (*File, bool) {
	fileStoreMu.Lock()
	defer fileStoreMu.Unlock()
	file, ok := fileStore[p]
	return file, ok
}

func Files() []*File {
	fileStoreMu.Lock()
	defer fileStoreMu.Unlock()

	keys := make([]string, 0, len(fileStore))
	for k := range fileStore {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	ret := make([]*File, len(fileStore))
	for i := range keys {
		ret[i] = fileStore[keys[i]]
	}

	return ret
}

func FileStore(p string, f *File) {
	fileStoreMu.Lock()
	defer fileStoreMu.Unlock()
	fileStore[p] = f
}

type Path struct {
	Dir    string
	Base   string
	File   string
	GoBase string
	Config *Config
}

func GetPathVar() string {
	if p := os.Getenv("MSIDLPATH"); p != "" {
		return p
	}
	return ""
}

func SetPathVar(p string) {
	os.Setenv("MSIDLPATH", p)
}

func ParsePathVar(p string) []*Path {
	var paths []*Path
	for _, dir := range strings.Split(p, ":") {
		if strings.Contains(dir, "=") {
			parts := strings.SplitN(dir, "=", 2)
			paths = append(paths, &Path{Dir: parts[1], GoBase: parts[0]})
		} else {
			paths = append(paths, &Path{Dir: dir})
		}
	}
	return paths
}

func FindPath() []*Path {
	var paths []*Path
	_, p, _, ok := runtime.Caller(0)
	if ok {
		paths = append(paths, &Path{Dir: filepath.Join(filepath.Dir(p), "idl")})
	}

	if p := GetPathVar(); p != "" {
		paths = append(paths, ParsePathVar(p)...)
	}

	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, &Path{Dir: wd})
	}

	return paths
}

func LookupType(n string) *Type {
	for _, f := range Files() {
		t, ok := f.exportSyms[n]
		if !ok || t.Type == nil {
			continue
		}
		return t.Type
	}
	return nil
}

func FindFile(f string) (*Path, error) {
	for _, p := range FindPath() {

		dir := p.Dir

		reldirs := []string{""}

		if idx := strings.LastIndex(f, "/"); idx > 0 {
			reldirs[0], f = f[:idx], f[idx+1:]
		}

		ps, err := filepath.Glob(filepath.Join(dir, "*"))
		if err != nil {
			return nil, err
		}

		for _, p := range ps {
			if info, _ := os.Stat(p); info != nil && info.IsDir() {
				reldirs = append(reldirs, filepath.Base(p))
			}
		}
		for _, reldir := range reldirs {
			dir := dir
			if reldir != "" {
				dir = filepath.Join(dir, reldir)
			}
			for file := range map[string]struct{}{
				strings.TrimPrefix(f, "ms-"): {},
				strings.TrimPrefix(f, "mc-"): {},
				f:                            {},
			} {
				path := filepath.Join(dir, file)
				if _, err := os.Stat(path); err != nil {
					continue
				}

				var cc *Config

				for _, dir := range []string{dir, p.Dir} {
					if cc, err = LoadConfig(dir); err == nil && cc != nil {
						break
					}
				}

				if err != nil {
					return nil, fmt.Errorf("cannot load config for file %q: %w", path, err)
				}

				if p.GoBase == "" && cc != nil {
					p.GoBase = cc.Go.Pkg
				}

				return &Path{
					File:   path,
					Base:   reldir,
					Dir:    p.Dir,
					GoBase: p.GoBase,
					Config: cc.For(path),
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("cannot find file %q: path: %s", f, GetPathVar())
}

func NewFile(p, pkg string) *File {

	if pkg == "" {
		pkg = /* "go-" + */ strings.TrimPrefix(strings.TrimPrefix(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)), "ms-"), "mc-")
	}

	pkg = strings.ReplaceAll(pkg, "-", "_")

	return &File{Path: p, GoPkg: pkg}
}

// File structure represents the parsed file contents.
type File struct {
	FullPath string `json:"full_path"`
	// Path is a path to file.
	Path string `json:"path"`
	// GoPkg ...
	GoPkg string `json:"go_pkg"`
	// GoPkgBase is the base name of the Go package.
	GoPkgBase string `json:"go_pkg_base,omitempty"`
	// MIDLConfig is the MIDL configuration for the file.
	MIDLConfig *Config `json:"-,omitempty"`
	// The list of imports.
	Imports []string `json:"imports,omitempty"`
	// Export is a map of exported symbols on file level.
	Export map[string]*Export `json:"exports,omitempty"`
	// Interfaces is a list of interfaces.
	Interfaces []*Interface `json:"interfaces,omitempty"`
	// ComClasses is a list of COM classes.
	ComClasses []*ComClass `json:"com_classes,omitempty"`
	// DispatchInterfaces ...
	DispatchInterfaces []*DispatchInterface `json:"dispatch_interfaces,omitempty"`
	// Libraries ...
	Libraries []*Library `json:"libraries,omitempty"`
	// exportSyms ...
	exportSyms map[string]*Export `json:"-"`
}

func (f *File) Namer() *go_names.Namer {
	if f.MIDLConfig != nil && f.MIDLConfig.Namer != nil {
		return f.MIDLConfig.Namer
	}
	return nil
}

// Exports function returns the exported symbols as a sorted array.
func (f *File) Exports() []*Export {

	exportSyms := make([]*Export, 0, len(f.Export))
	for _, sym := range f.Export {
		exportSyms = append(exportSyms, sym)
	}

	sort.Slice(exportSyms, func(i, j int) bool {
		return exportSyms[i].Position < exportSyms[j].Position
	})

	return exportSyms
}

// LookupType function lookups the type inside the file.
func (f *File) LookupType(n string) (*Type, bool) {

	if typ, ok := f.Export[n]; ok && typ.Type != nil {
		return typ.Type, true
	}

	for i := range f.Interfaces {
		if typ, ok := f.Interfaces[i].Body.Export[n]; ok && typ.Type != nil {
			return typ.Type, true
		}
	}

	return nil, false
}

func (f *File) LookupAlias(n string) []string {

	if typ, ok := f.Export[n]; ok {
		return typ.Aliases
	}

	for i := range f.Interfaces {
		if typ, ok := f.Interfaces[i].Body.Export[n]; ok {
			return typ.Aliases
		}
	}
	return nil
}

func (f *File) IsLocal(tn string) bool {
	if _, ok := f.exportSyms[tn]; ok {
		return true
	}
	return false
}

// GoPackage function returns the go package name for the type.
func (f *File) GoPackage(tn string) (string, bool) {

	if _, ok := f.exportSyms[tn]; ok {
		return "", true
	}

	for _, file := range Files() {
		if _, ok := file.exportSyms[tn]; ok {
			return file.GoPkg, true
		}
	}

	return "", false
}

func (f *File) IsDCOM() bool {
	for _, iff := range f.Interfaces {
		if iff.IsObject() {
			return true
		}
	}
	return false
}

func (f *File) PkgIs(pkg string) bool {
	return f.GoPkg == pkg
}
