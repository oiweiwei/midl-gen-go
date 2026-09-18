package midl

import "github.com/oiweiwei/midl-gen-go/midl/uuid"

type Library struct {
	Name  string       `json:"name"`
	Attrs *LibraryAttr `json:"attr,omitempty"`
	Body  *File        `json:"body,omitempty"`
}

type ImportLib struct {
	Name string `json:"name"`
}

// ComClass structure ...
type ComClass struct {
	Name       string          `json:"name,omitempty"`
	Attrs      *ComClassAttr   `json:"attrs,omitempty"`
	Interfaces []*ComInterface `json:"interfaces,omitempty"`
}

// ComInterface structure ...
type ComInterface struct {
	Name  string            `json:"name,omitempty"`
	Type  *Type             `json:"type,omitempty"`
	Attrs *ComInterfaceAttr `json:"attrs,omitempty"`
}

// DispatchInterface ...
type DispatchInterface struct {
	// The dispatch interface name.
	Name string `json:"name"`
	// The dispatch interface attributes.
	Attrs *DispatchInterfaceAttr `json:"attr,omitempty"`
	// The dispatch interface body.
	Body DispatchInterfaceBody `json:"body,omitempty"`
}

// DispatchInterfaceBody ...
type DispatchInterfaceBody struct {
	// The dispatch interface properties.
	Properties []*Field `json:"properties,omitempty"`
	// The dispatch interfaces methods.
	Methods []*Operation `json:"methods,omitempty"`
}

// ComClassAttr ...
type ComClassAttr struct {
	*InterfaceAttr
	AppObject bool
}

// DispatchInterfaceAttr ...
type DispatchInterfaceAttr struct {
	*InterfaceAttr
}

// LibraryAttr ...
type LibraryAttr struct {
	*InterfaceAttr
}

// ComInterfaceAttr ...
type ComInterfaceAttr struct {
	Default bool
	Source  bool
}

type Module struct {
	Name    string          `json:"name,omitempty"`
	Attrs   *ModuleAttr     `json:"attrs,omitempty"`
	Members []*ModuleMember `json:"members,omitempty"`
}

func (m *Module) HasConstants() bool {
	for _, m := range m.Members {
		if m.Const != nil {
			return true
		}
	}
	return false
}

type ModuleAttr struct {
	UUID        *uuid.UUID
	Version     *Version
	HelpString  string
	HelpContext string
	Hidden      bool
	DLLName     string
}

type ModuleMember struct {
	Const *Const     `json:"const,omitempty"`
	Func  *Operation `json:"func,omitempty"`
}
