package midl

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	go_names "github.com/oiweiwei/midl-gen-go/gonames"
	"gopkg.in/yaml.v3"
)

// Config is the configuration for midl and codegen.
type Config struct {
	Go struct {
		Pkg string `yaml:"pkg,omitempty"`
	} `yaml:"go,omitempty"`
	// Path is the path for gonames configuration.
	GoNames struct {
		Path   string                        `yaml:"path,omitempty"`
		Global *go_names.Config              `yaml:"global,omitempty"`
		IDL    []map[string]*go_names.Config `yaml:"idl,omitempty"`
	} `yaml:"gonames,omitempty"`
	// Namer is the namer to use for code generation.
	Namer *go_names.Namer `yaml:"-"`
}

func (c *Config) For(idl string) *Config {

	if c == nil {
		return nil
	}

	if c.Namer != nil {
		return c
	}

	n := &go_names.Config{}

	idl = filepath.Base(idl)
	idl = strings.TrimSuffix(idl, path.Ext(idl))

	n = n.Merge(c.GoNames.Global)

	for _, m := range c.GoNames.IDL {
		if cfg, ok := m[idl]; ok {
			n = n.Merge(cfg)
			break
		}
	}

	cc := *c
	cc.Namer = &go_names.Namer{Config: n}
	return &cc
}

var (
	configsStoreMu = &sync.RWMutex{}
	configsStore   = make(map[string]*Config)
)

const ConfigDir = ".midl"

func LoadConfig(dir string) (*Config, error) {
	configsStoreMu.RLock()
	if c, ok := configsStore[dir]; ok {
		configsStoreMu.RUnlock()
		return c, nil
	}
	configsStoreMu.RUnlock()

	fmt.Println("loading config for dir", dir)

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

	c := &Config{}

	b, err := os.ReadFile(filepath.Join(dir, ConfigDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("load config: read config.yaml: %w", err)
	}

	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("load config: unmarshal config.yaml: %w", err)
	}

	if c.GoNames.Path != "" {
		if c.Namer, err = go_names.LoadNamerFromPath(filepath.Join(dir, ConfigDir, c.GoNames.Path)); err != nil {
			return nil, fmt.Errorf("load config: load gonames: %w", err)
		}
	}

	configsStoreMu.Lock()
	configsStore[dir] = c
	configsStoreMu.Unlock()

	return c, nil
}
