package openspecs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

// index.go contains the implementation of the Index type, which represents the index of a
// specific protocol in the Microsoft Open Specifications.

// Index represents the index of a specific protocol in the Microsoft Open Specifications.
type Index map[string]map[string]string

func (p *Index) Unmarshal(ctx context.Context, r io.Reader) error {
	return json.NewDecoder(r).Decode(p)
}

func (p Index) Marshal(ctx context.Context) (io.Reader, error) {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

var _ Cacher = (*Index)(nil)

// Add adds an entry to the Index with the given name and URL.
func (p Index) Add(entry, name, url string) {
	if p[entry] == nil {
		p[entry] = make(map[string]string)
	}
	p[entry][name] = url
}

// GetAny retrieves any URL associated with the given entry name. It returns the URL and a
// boolean indicating whether an entry was found.
func (p Index) GetAny(entry string) (string, bool) {
	for _, url := range p[entry] {
		return url, true
	}
	return "", false
}

// EachSelf iterates over each entry in the Index, calling the provided function with the entry
// name and its associated URLs. If the function returns false, the iteration stops.
func (p Index) EachSelf(f func(string, string) bool) {
	for name, values := range p {
		if value, ok := values[name]; ok {
			if !f(name, value) {
				return
			}
		}
	}
}

// Each iterates over each entry in the Index, calling the provided function with the entry
// name and its associated URLs. If the function returns false, the iteration stops.
func (p Index) Each(f func(string, map[string]string) bool) {
	for entry, values := range p {
		if !f(entry, values) {
			return
		}
	}
}

var extractNameRe = regexp.MustCompile(
	`^(?:[A-Za-z_0-9]+::)?` +
		`_?([A-Za-z_0-9]+)` +
		`(?:\s[mM]ethod)?` +
		`(?:\s\(get\))?` +
		`(?:\s\([Oo]pnum:?\s?\d+\))?\s?` +
		`(?:[mM]ethod|[sS]tructure|Structurestructure|[eE]numeration|[pP]acket)$`)

// ExtractName attempts to extract a valid name from the given string. It returns the extracted name
// and a boolean indicating whether a valid name was successfully extracted. The function uses a
// regular expression to match common patterns in protocol documentation, such as method names,
// structure names, and enumeration names. It also checks for the presence of "Introduction" or
// "Overview" in the name, which are considered valid entries.
func ExtractName(name string) (string, bool) {

	name = strings.TrimSpace(strings.ReplaceAll(name, "\n", ""))

	if matches := extractNameRe.FindStringSubmatch(name); len(matches) > 1 {
		return matches[1], true
	}

	if strings.Contains(name, "Introduction") || strings.Contains(name, "Overview") {
		return name, true
	}

	return "", false
}
