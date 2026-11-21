package gonames

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the naming rules as written in YAML, see README.md for the
// meaning of each section. The zero Config has no rules.
type Config struct {
	// Rename maps a whole identifier to the words of its Go name.
	Rename map[string][]string `yaml:"rename"`
	// Override renames one member of one owner: the owner is the IDL name of a
	// struct, union, enum or operation, the member the IDL name of its field,
	// parameter or element, and the value the words of the Go name. It applies to
	// that member only and takes precedence over every other rule.
	Override map[string]map[string][]string `yaml:"override"`
	// Rotate maps a leading word to the word that is moved to the end.
	Rotate map[string]string `yaml:"rotate"`
	// Abbr maps a lower case word to the exact spelling used in the Go name.
	Abbr map[string]string `yaml:"abbr"`
	// Acronyms are spellings of words that are written as they are listed
	// (ACL, DWORD, IPv6): a word that equals one ignoring case becomes that
	// spelling. It is the short form of an abbr entry whose value is its own key
	// in another case; Abbr entries take precedence. Lexems are not joined:
	// joining I and Pv6 into IPv6 would also change what a rule such as a
	// trimmed prefix sees (pIn must stay In, not PIN) and the words of a snake
	// case name (nt_owf stays two words).
	Acronyms []string `yaml:"acronyms"`
	// Split maps a lexem to the words it consists of.
	Split map[string][]string `yaml:"split"`
	// LookBehind maps a previous lexem and the current one to the words that
	// replace both.
	LookBehind map[string]map[string][]string `yaml:"lookBehind"`
	Trim       Trim                           `yaml:"trim"`
}

// Trim is the part of the rules that drops text.
type Trim struct {
	// Words are dropped wherever they appear (lower case).
	Words []string `yaml:"word"`
	// Prefixes are dropped when they are the first word.
	Prefixes []string `yaml:"prefix"`
	// PrefixAll are removed from the identifier before it is split.
	PrefixAll []string `yaml:"prefixAll"`
	// Suffix are removed from the identifier before it is split.
	Suffix []string `yaml:"suffix"`
}

// Merge returns the rules of c overridden and extended by the rules of other.
// Neither c nor other is modified. Map entries of other replace those of c;
// lists are concatenated.
func (c *Config) Merge(other *Config) *Config {

	out := &Config{
		Rename:     map[string][]string{},
		Override:   map[string]map[string][]string{},
		Rotate:     map[string]string{},
		Abbr:       map[string]string{},
		Split:      map[string][]string{},
		LookBehind: map[string]map[string][]string{},
	}

	for _, src := range []*Config{c, other} {
		if src == nil {
			continue
		}
		for k, v := range src.Rename {
			out.Rename[k] = append([]string(nil), v...)
		}
		for owner, members := range src.Override {
			m := out.Override[owner]
			if m == nil {
				m = map[string][]string{}
				out.Override[owner] = m
			}
			for member, words := range members {
				m[member] = append([]string(nil), words...)
			}
		}
		for k, v := range src.Rotate {
			out.Rotate[k] = v
		}
		for k, v := range src.Abbr {
			out.Abbr[k] = v
		}
		for k, v := range src.Split {
			out.Split[k] = append([]string(nil), v...)
		}
		for k, v := range src.LookBehind {
			m := out.LookBehind[k]
			if m == nil {
				m = map[string][]string{}
				out.LookBehind[k] = m
			}
			for k2, v2 := range v {
				m[k2] = append([]string(nil), v2...)
			}
		}
		out.Acronyms = append(out.Acronyms, src.Acronyms...)
		out.Trim.Words = append(out.Trim.Words, src.Trim.Words...)
		out.Trim.Prefixes = append(out.Trim.Prefixes, src.Trim.Prefixes...)
		out.Trim.PrefixAll = append(out.Trim.PrefixAll, src.Trim.PrefixAll...)
		out.Trim.Suffix = append(out.Trim.Suffix, src.Trim.Suffix...)
	}

	return out
}

// ParseConfig parses the rules from YAML.
func ParseConfig(b []byte) (*Config, error) {
	c := &Config{}
	if err := yaml.NewDecoder(bytes.NewReader(b)).Decode(c); err != nil && !errors.Is(err, io.EOF) { // an empty document has no rules.
		return nil, fmt.Errorf("gonames: parse config: %w", err)
	}
	return c, nil
}

// LoadConfigFromPath loads the rules from all *.yaml files of the directory, in
// name order, and merges them: a map entry of a later file replaces that of an
// earlier one, lists are concatenated. A directory without files has no rules.
func LoadConfigFromPath(dir string) (*Config, error) {

	names, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("gonames: load config: %w", err)
	}

	cfg := &Config{}

	for _, name := range names {
		b, err := os.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("gonames: load config: %w", err)
		}
		c, err := ParseConfig(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		cfg = cfg.Merge(c)
	}

	return cfg, nil
}

// LoadNamerFromPath loads the rules from the directory (see LoadConfigFromPath)
// and returns the namer.
func LoadNamerFromPath(dir string) (*Namer, error) {
	c, err := LoadConfigFromPath(dir)
	if err != nil {
		return nil, err
	}
	return NewNamer(c), nil
}

//go:embed default.yaml
var defaultYAML []byte

// defaultNamer is the namer without per-IDL rules: the built-in defaults only.
var defaultNamer = func() *Namer {
	c, err := ParseConfig(defaultYAML)
	if err != nil {
		panic(err) // the embedded file is part of the package.
	}
	return NewNamer(c)
}()
