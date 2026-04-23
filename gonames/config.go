package go_names

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

var defaultConfig *Config

func init() { defaultConfig = LoadConfig() }

type Config struct {
	Rename     map[string][]string            `yaml:"rename"`
	Rotate     map[string]string              `yaml:"rotate"`
	Abbr       map[string]string              `yaml:"abbr"`
	Split      map[string][]string            `yaml:"split"`
	LookBehind map[string]map[string][]string `yaml:"lookBehind"`
	Trim       struct {
		Words     []string            `yaml:"word"`
		Word      map[string]struct{} `yaml:"-"`
		PrefixAll []string            `yaml:"prefixAll"`
		Prefixes  []string            `yaml:"prefix"`
		Prefix    map[string]struct{} `yaml:"-"`
		Suffix    []string            `yaml:"suffix"`
	}
}

func LoadConfig() *Config {
	_, p, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot determine current dir")
	}
	c, err := LoadConfigFromPath(filepath.Join(filepath.Dir(p), "config"))
	if err != nil {
		panic(fmt.Errorf("gonames: load config: %w", err))
	}
	return c
}

func LoadNamerFromPath(p string) (*Namer, error) {

	c, err := LoadConfigFromPath(p)
	if err != nil {
		panic(fmt.Errorf("gonames: load config: %w", err))
	}
	if c == nil {
		return nil, nil
	}
	return &Namer{c}, nil

}

func (c *Config) Copy() *Config {
	c2 := &Config{}

	c2.Rename = make(map[string][]string)
	for k, v := range c.Rename {
		c2.Rename[k] = append([]string(nil), v...)
	}

	c2.Rotate = make(map[string]string)
	for k, v := range c.Rotate {
		c2.Rotate[k] = v
	}

	c2.Abbr = make(map[string]string)
	for k, v := range c.Abbr {
		c2.Abbr[k] = v
	}

	c2.Split = make(map[string][]string)
	for k, v := range c.Split {
		c2.Split[k] = append([]string(nil), v...)
	}

	c2.LookBehind = make(map[string]map[string][]string)
	for k, v := range c.LookBehind {
		c2.LookBehind[k] = make(map[string][]string)
		for k2, v2 := range v {
			c2.LookBehind[k][k2] = append([]string(nil), v2...)
		}
	}

	c2.Trim.PrefixAll = append([]string(nil), c.Trim.PrefixAll...)
	c2.Trim.Prefixes = append([]string(nil), c.Trim.Prefixes...)
	c2.Trim.Suffix = append([]string(nil), c.Trim.Suffix...)

	for k := range c.Trim.Prefix {
		c2.Trim.Prefix[k] = struct{}{}
	}

	for k := range c.Trim.Word {
		c2.Trim.Word[k] = struct{}{}
	}

	return c2
}

func (c *Config) Merge(other *Config) *Config {

	c2 := c.Copy()

	if other == nil {
		return c2
	}

	for k, v := range other.Rename {
		c2.Rename[k] = v
	}

	for k, v := range other.Rotate {
		c2.Rotate[k] = v
	}

	for k, v := range other.Abbr {
		c2.Abbr[k] = v
	}

	for k, v := range other.Split {
		c2.Split[k] = v
	}

	for k, v := range other.LookBehind {
		c2.LookBehind[k] = v
	}

	c2.Trim.PrefixAll = append(c2.Trim.PrefixAll, other.Trim.PrefixAll...)
	c2.Trim.Prefixes = append(c2.Trim.Prefixes, other.Trim.Prefixes...)
	c2.Trim.Suffix = append(c2.Trim.Suffix, other.Trim.Suffix...)

	for k := range other.Trim.Prefix {
		if c2.Trim.Prefix == nil {
			c2.Trim.Prefix = make(map[string]struct{})
		}
		c2.Trim.Prefix[k] = struct{}{}
	}

	for k := range other.Trim.Word {
		if c2.Trim.Word == nil {
			c2.Trim.Word = make(map[string]struct{})
		}
		c2.Trim.Word[k] = struct{}{}
	}

	return c2
}

// ParseConfig parses the YAML configuration from the provided byte slice and returns a Config struct.
func ParseConfig(b []byte) (*Config, error) {
	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("gonames: parse config: unmarshal yaml: %w", err)
	}
	c.Trim.Prefix = make(map[string]struct{})
	c.Trim.Word = make(map[string]struct{})
	for _, pfx := range c.Trim.Prefixes {
		c.Trim.Prefix[pfx] = struct{}{}
	}
	for _, wrd := range c.Trim.Words {
		c.Trim.Word[wrd] = struct{}{}
	}
	return c, nil
}

// UnmarshalYAML implements yaml.Unmarshaler to populate the Prefix and Word maps after decoding the YAML.
func (c *Config) UnmarshalYAML(n *yaml.Node) error {

	type config Config
	if err := n.Decode((*config)(c)); err != nil {
		return fmt.Errorf("gonames: unmarshal config: decode yaml: %w", err)
	}

	c.Trim.Prefix = make(map[string]struct{})
	c.Trim.Word = make(map[string]struct{})

	for _, pfx := range c.Trim.Prefixes {
		c.Trim.Prefix[pfx] = struct{}{}
	}
	for _, wrd := range c.Trim.Words {
		c.Trim.Word[wrd] = struct{}{}
	}

	return nil
}

// LoadConfigFromPath loads the configuration from all YAML files in the specified directory
// and merges them into a single Config struct.
func LoadConfigFromPath(p string) (*Config, error) {

	cc := Config{}

	fileNames, _ := filepath.Glob(filepath.Join(p, "*.yaml"))

	for _, fileName := range fileNames {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("gonames: load config: open file: %w", err)
		}
		if err := yaml.NewDecoder(f).Decode(&cc); err != nil {
			return nil, fmt.Errorf("gonames: load config: decode yaml: %w", err)
		}
	}

	return &cc, nil
}
