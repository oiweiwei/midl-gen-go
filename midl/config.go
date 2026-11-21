package midl

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oiweiwei/midl-gen-go/gonames"
	"gopkg.in/yaml.v3"
)

// Config is the configuration for midl and codegen.
type Config struct {
	Go struct {
		Pkg string `yaml:"pkg,omitempty"`
	} `yaml:"go,omitempty"`
	// Path is the path for gonames configuration.
	GoNames struct {
		Path   string                       `yaml:"path,omitempty"`
		Global *gonames.Config              `yaml:"global,omitempty"`
		IDL    []map[string]*gonames.Config `yaml:"idl,omitempty"`
	} `yaml:"gonames,omitempty"`
	// Namer is the namer to use for code generation.
	Namer *gonames.Namer `yaml:"-"`

	namersMu sync.Mutex
	namers   map[string]*gonames.Namer
}

// For returns the configuration for the IDL file: the namer is the global one
// extended with the rules written for that file.
func (c *Config) For(idl string) *Config {

	if c == nil {
		return nil
	}

	idl = filepath.Base(idl)
	idl = strings.TrimSuffix(idl, path.Ext(idl))

	for _, m := range c.GoNames.IDL {
		if rules, ok := m[idl]; ok {
			return &Config{Namer: c.NamerFor(idl, rules)}
		}
	}

	if c.Namer != nil {
		return c
	}

	return &Config{Namer: gonames.NewNamer(nil)}
}

// NamerFor returns the namer of the IDL: the global rules extended with the
// rules of the file. The namer is built once per IDL.
func (c *Config) NamerFor(idl string, rules *gonames.Config) *gonames.Namer {

	c.namersMu.Lock()
	defer c.namersMu.Unlock()

	if nr, ok := c.namers[idl]; ok {
		return nr
	}

	if c.namers == nil {
		c.namers = make(map[string]*gonames.Namer)
	}

	nr := gonames.NewNamer(c.Namer.Config().Merge(rules))
	c.namers[idl] = nr

	return nr
}

const ConfigDir = ".midl"

// LoadConfig loads (once per session) the .midl/config.yaml of the directory.
// It returns nil, nil if the directory has no .midl folder.
func (s *Session) LoadConfig(dir string) (*Config, error) {
	s.mu.Lock()
	c, ok := s.configs[dir]
	s.mu.Unlock()
	if ok {
		return c, nil
	}

	s.log.Debug("loading config", "dir", dir)

	stat, err := os.Stat(filepath.Join(dir, ConfigDir))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load config: stat .midl folder: %w", err)
		}
		return nil, nil
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("load config: .midl is not a directory")
	}

	c = &Config{}

	b, err := os.ReadFile(filepath.Join(dir, ConfigDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("load config: read config.yaml: %w", err)
	}

	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("load config: unmarshal config.yaml: %w", err)
	}

	if c.GoNames.Path != "" {
		if c.Namer, err = gonames.LoadNamerFromPath(filepath.Join(dir, ConfigDir, c.GoNames.Path)); err != nil {
			return nil, fmt.Errorf("load config: load gonames: %w", err)
		}
	}

	if c.GoNames.Global != nil {
		c.Namer = gonames.NewNamer(c.Namer.Config().Merge(c.GoNames.Global))
	}

	s.mu.Lock()
	s.configs[dir] = c
	s.mu.Unlock()

	return c, nil
}
