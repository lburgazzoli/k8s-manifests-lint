package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type SourceType string

const (
	SourceTypeYAML       SourceType = "yaml"
	SourceTypeHelm       SourceType = "helm"
	SourceTypeKustomize  SourceType = "kustomize"
	SourceTypeGoTemplate SourceType = "gotemplate"
	SourceTypeTemplate   SourceType = "template"
)

func (s SourceType) String() string {
	return string(s)
}

func (s SourceType) IsValid() bool {
	switch s {
	case SourceTypeYAML, SourceTypeHelm, SourceTypeKustomize, SourceTypeGoTemplate, SourceTypeTemplate, "":
		return true
	default:
		return false
	}
}

type Config struct {
	Sources []Source      `mapstructure:"sources"`
	Linters LintersConfig `mapstructure:"linters"`
	Output  OutputConfig  `mapstructure:"output"`
	Exclude ExcludeConfig `mapstructure:"exclude"`
	Run     RunConfig     `mapstructure:"run"`
}

type Source struct {
	Type   SourceType             `mapstructure:"type"`
	Path   string                 `mapstructure:"path"`
	Chart  string                 `mapstructure:"chart"`
	Values string                 `mapstructure:"values"`
	Data   map[string]interface{} `mapstructure:"data"`
}

type LintersConfig struct {
	Enable   []string                          `mapstructure:"enable"`
	Disable  []string                          `mapstructure:"disable"`
	Settings map[string]map[string]interface{} `mapstructure:"settings"`
}

type OutputConfig struct {
	Format     string `mapstructure:"format"`
	ShowSource bool   `mapstructure:"show-source"`
	Color      string `mapstructure:"color"`
}

type ExcludeConfig struct {
	Resources []ResourceFilter `mapstructure:"resources"`
	Paths     []string         `mapstructure:"paths"`
}

type ResourceFilter struct {
	Kind      string `mapstructure:"kind"`
	Name      string `mapstructure:"name"`
	Namespace string `mapstructure:"namespace"`
}

type RunConfig struct {
	SkipDirs []string `mapstructure:"skip-dirs"`
}

func Load(configFile string) (*Config, error) {
	v := viper.New()

	v.SetDefault("output.format", "text")
	v.SetDefault("output.show-source", true)
	v.SetDefault("output.color", "auto")
	v.SetDefault("run.timeout", "5m")

	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		v.SetConfigName(".k8s-manifests-lint")
		v.SetConfigType("yaml")
		v.AddConfigPath(cwd)
		v.AddConfigPath(filepath.Join(cwd, ".config"))
		v.AddConfigPath(".")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	validFormats := map[string]bool{
		"text":           true,
		"json":           true,
		"yaml":           true,
		"github-actions": true,
		"sarif":          true,
	}

	if !validFormats[c.Output.Format] {
		return fmt.Errorf("invalid output format: %s", c.Output.Format)
	}

	for i, source := range c.Sources {
		if !source.Type.IsValid() {
			return fmt.Errorf("invalid source type at index %d: %s", i, source.Type)
		}
	}

	return nil
}
