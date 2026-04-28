package openspecs

// indexer.go contains the implementation of the ProtocolIndexer, which manages a
// collection of protocol indexes.

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ProtocolIndex represents an entry in the protocol index, containing the protocol family,
// name, and UUID.
type ProtocolIndex struct {
	// Protocol family, e.g. "windows_protocols".
	Family string `yaml:"family" json:"family"`
	// Name of the entry, e.g. "ms-dnsp", "dnsp", or "mc-ccfg".
	Name string `yaml:"name" json:"name"`
	// UUID of the entry, e.g. "b1c9e5c8-9f3a-4d2e-8c1a-2f3e4d5f6a7b".
	UUID string `yaml:"uuid" json:"uuid"`
	// Ref is the reference to another protocol.
	Ref string `yaml:"ref,omitempty" json:"ref,omitempty"`
}

type Reference struct {
	Name    string   `yaml:"name" json:"name"`
	UUID    string   `yaml:"uuid" json:"uuid"`
	Aliases []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
}

type Extra []map[string][]*Reference

func ReadExtraFromFile(path string) (Extra, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("protocol index: read extra from file: open file: %w", err)
	}

	defer f.Close()

	return ReadExtraFrom(f)
}

func ReadExtraFrom(r io.Reader) (Extra, error) {
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	extra := make(Extra, 0)

	if err := decoder.Decode(&extra); err != nil {
		return nil, fmt.Errorf("protocol index: load extra from reader: decode yaml: %w", err)
	}

	return extra, nil
}

// ProtocolIndexer represents the index of indexes.
type ProtocolIndexer struct {
	mu sync.RWMutex
	// The list of protocol indexes.
	i []*ProtocolIndex
	// lookup maps protocol family and name to the index of the protocol in the list.
	l map[string]int
	// extra contains additional information about protocols, such as aliases and UUIDs.
	extra []map[string][]*Reference
}

func NewProtocolIndexerFromFile(path string) (*ProtocolIndexer, error) {
	p := NewProtocolIndexer()
	if err := p.ReadFromFile(path); err != nil {
		return nil, err
	}
	return p, nil
}

func NewProtocolIndexer() *ProtocolIndexer {
	return &ProtocolIndexer{
		i: make([]*ProtocolIndex, 0),
		l: make(map[string]int),
	}
}

// ReadFromFile reads the protocol index from a YAML file. It opens the specified file, creates a reader,
// and calls the ReadFrom method to load the protocol index. If any error occurs during file opening or
// reading, it returns an error.
func (p *ProtocolIndexer) ReadFromFile(path string) error {

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("protocol index: read from file: open file: %w", err)
	}

	defer f.Close()

	return p.ReadFrom(f)
}

// ProtocolIndexer implements the io.ReaderFrom interface, allowing it to be loaded from any reader,
// such as a file or a network response. It decodes the YAML content into a slice of ProtocolIndex
// structs and adds each protocol to the indexer using the Add method. If any protocol fails to add
// (e.g., due to duplicates), it returns an error.
func (p *ProtocolIndexer) ReadFrom(r io.Reader) error {

	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	ps := make([]*ProtocolIndex, 0)

	if err := decoder.Decode(&ps); err != nil {
		return fmt.Errorf("protocol index: load from reader: decode yaml: %w", err)
	}

	for _, protocol := range ps {
		if !p.Add(protocol) {
			return fmt.Errorf("protocol index: load from reader: add protocol: protocol %s/%s already exists", protocol.Family, protocol.Name)
		}
	}

	return nil
}

func (p *ProtocolIndexer) ReadExtraFrom(r io.Reader) error {
	e, err := ReadExtraFrom(r)
	if err != nil {
		return err
	}
	p.extra = e
	return nil
}

func (p *ProtocolIndexer) ReadExtraFromFile(path string) error {
	e, err := ReadExtraFromFile(path)
	if err != nil {
		return err
	}
	p.extra = e
	return nil
}

// Add adds a protocol index to the indexer. It normalizes the protocol name and family to lowercase and
// handles common prefixes like "ms-" and "mc-". It ensures that each protocol is only added once, even
// if it can be accessed through multiple keys.
func (p *ProtocolIndexer) Add(protocol *ProtocolIndex) bool {

	p.mu.Lock()
	defer p.mu.Unlock()

	name, family, last := strings.ToLower(protocol.Name), strings.ToLower(protocol.Family), len(p.i)

	if family == "" {
		// default to "windows_protocols" if family is not specified.
		protocol.Family = WindowsProtocols
	}

	// normalize the UUID to lowercase and extract the reference if it contains a "/".
	if protocol.UUID = strings.ToLower(protocol.UUID); protocol.UUID != "" {
		if strings.Contains(protocol.UUID, "/") {
			protocol.Ref, protocol.UUID, _ = strings.Cut(protocol.UUID, "/")
		}
	}

	if !strings.Contains(name, "ms-") && !strings.Contains(name, "mc-") {
		// if the name does not contain "ms-" or "mc-", add "ms-" prefix by default.
		protocol.Name = "ms-" + name
	}

	if strings.HasPrefix(name, "ms-") || strings.HasPrefix(name, "mc-") {
		// if the name starts with "ms-" or "mc-", also add a version without the prefix for easier access.
		name = name[3:]
	}

	keys := []string{
		name,
		"ms-" + name,
		"mc-" + name,
		family + "/" + name,
		family + "/ms-" + name,
		family + "/mc-" + name,
	}

	if protocol.Ref == "" && protocol.UUID != "" {
		keys = append(keys, protocol.UUID)
	}

	for _, key := range keys {
		if index, ok := p.l[key]; ok && index != last {
			return false // already exists
		}
	}

	p.i = append(p.i, protocol)

	for _, key := range keys {
		p.l[key] = last
	}

	return true
}

func (p *ProtocolIndexer) GetExtra(protocolName string) []*Reference {

	if p.extra == nil {
		return nil
	}

	name := strings.ToLower(protocolName)
	if strings.Contains(name, ".") {
		// if the name contains a ".", remove the extension for lookup, e.g. "ms-dnsp.idl" -> "ms-dnsp".
		name = name[:strings.LastIndex(name, ".")]
	}
	if strings.Contains(name, "/") {
		// if the name contains a "/", remove the family for lookup, e.g. "windows_protocols/ms-dnsp" -> "ms-dnsp".
		name = name[strings.LastIndex(name, "/")+1:]
	}
	if strings.HasPrefix(name, "ms-") || strings.HasPrefix(name, "mc-") {
		// if the name starts with "ms-" or "mc-", also check for a version without the prefix for easier access.
		name = name[3:]
	}

	for _, extra := range p.extra {
		if refs, ok := extra[name]; ok {
			return refs
		}
	}

	return nil
}

// Get retrieves a protocol index by its name. It normalizes the protocol name to lowercase and checks for
// common prefixes like "ms-" and "mc-". It returns the protocol index and a boolean indicating whether it was found.
func (p *ProtocolIndexer) Get(protocolName string) (*ProtocolIndex, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	name := strings.ToLower(protocolName)
	if strings.Contains(name, ".") {
		// if the name contains a ".", remove the extension for lookup, e.g. "ms-dnsp.idl" -> "ms-dnsp".
		name = name[:strings.LastIndex(name, ".")]
	}

	if index, ok := p.l[strings.ToLower(protocolName)]; ok {
		// iteratively resolve references until we find a protocol without a reference.
		for p.i[index].Ref != "" {
			if index, ok = p.l[p.i[index].Ref]; !ok {
				return nil, false
			}
		}
		return p.i[index], true
	}

	return nil, false
}
