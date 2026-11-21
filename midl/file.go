package midl

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/oiweiwei/midl-gen-go/gonames"
)

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
	MIDLConfig *Config `json:"-"`
	// sess is the session the file was loaded in.
	sess *Session
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
	// ImportLibs ...
	ImportLibs []*ImportLib `json:"import_libs,omitempty"`
	// Modules ...
	Modules []*Module `json:"modules,omitempty"`
	// Libraries ...
	Libraries []*Library `json:"libraries,omitempty"`
	// exportSyms ...
	exportSyms map[string]*Export `json:"-"`
}

func (f *File) IsEmpty() bool {

	for _, e := range f.Export {
		if e.Const != nil || e.Type != nil {
			return false
		}
	}

	if len(f.ComClasses) > 0 || len(f.Libraries) > 0 {
		return false
	}

	for _, iff := range f.Interfaces {
		if !iff.ForwardDeclarator {
			return false
		}
	}

	for _, m := range f.Modules {
		if m.HasConstants() {
			return false
		}
	}

	return true
}

func (f *File) Namer() *gonames.Namer {
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

	for _, file := range f.Session().Files() {
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

// Session returns the session the file was loaded in.
func (f *File) Session() *Session {
	return f.sess
}
