package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lint/pkg/config"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/linter"
	_ "github.com/lburgazzoli/k8s-manifests-lint/pkg/linters"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/output"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/renderer"
	"github.com/lburgazzoli/k8s-manifests-lint/pkg/renderer/yaml"
)

var (
	cfgFile        string
	enableLinters  []string
	disableLinters []string
	outputFormat   string
	noColor        bool
	failOnWarning  bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "k8s-manifests-lint",
	Short: "A pluggable linter for Kubernetes manifests",
	Long: `k8s-manifests-lint is a pluggable linter for Kubernetes manifests inspired by golangci-lint.
It provides a unified interface for running multiple linters against Kubernetes resources.`,
	SilenceUsage: true,
}

var runCmd = &cobra.Command{
	Use:   "run [path...]",
	Short: "Run linters on Kubernetes manifests",
	RunE:  runLint,
}

var lintersCmd = &cobra.Command{
	Use:   "linters",
	Short: "List all available linters",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, l := range linter.All() {
			fmt.Printf("%-30s %s\n", l.Name(), l.Description())
		}
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Println("Configuration is valid")
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate example configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		example := `.k8s-manifests-lint.yaml created with example configuration.
See documentation for all available options.`
		fmt.Println(example)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("k8s-manifests-lint version 0.1.0")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: .k8s-manifests-lint.yaml)")
	rootCmd.PersistentFlags().StringSliceVar(&enableLinters, "enable-linter", nil, "enable specific linter(s)")
	rootCmd.PersistentFlags().StringSliceVar(&disableLinters, "disable-linter", nil, "disable specific linter(s)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "text", "output format (text|json|yaml|github-actions|sarif)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&failOnWarning, "fail-on-warning", false, "exit with error on warnings")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(lintersCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	var allObjects []unstructured.Unstructured

	if len(cfg.Sources) > 0 {
		for _, source := range cfg.Sources {
			r, err := renderer.NewFromSource(source)
			if err != nil {
				return fmt.Errorf("failed to create renderer for source type %q: %w", source.Type, err)
			}

			path := source.Path
			if path == "" {
				path = "."
			}

			objects, err := r.Render(cmd.Context(), path)
			if err != nil {
				return fmt.Errorf("failed to render manifests from source (type: %s, path: %s): %w", source.Type, path, err)
			}
			allObjects = append(allObjects, objects...)
		}
	} else {
		paths := args
		if len(paths) == 0 {
			paths = []string{"."}
		}

		r := yaml.New(config.Source{})
		for _, path := range paths {
			objects, err := r.Render(cmd.Context(), path)
			if err != nil {
				return fmt.Errorf("failed to render manifests from %q: %w", path, err)
			}
			allObjects = append(allObjects, objects...)
		}
	}

	enabledLinters := cfg.Linters.Enable
	if len(enableLinters) > 0 {
		enabledLinters = enableLinters
	}

	disabledLinters := cfg.Linters.Disable
	if len(disableLinters) > 0 {
		disabledLinters = disableLinters
	}

	runner, err := linter.NewRunner(&linter.RunnerConfig{
		EnabledLinters:  enabledLinters,
		DisabledLinters: disabledLinters,
		Settings:        cfg.Linters.Settings,
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	issues, err := runner.Run(cmd.Context(), allObjects)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	format := cfg.Output.Format
	if outputFormat != "text" {
		format = outputFormat
	}

	useColor := !noColor && cfg.Output.Color != "never"
	formatter, err := output.NewFormatter(format, useColor)
	if err != nil {
		return err
	}

	if err := formatter.Format(os.Stdout, issues); err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fatalCount := 0
	errorCount := 0
	warningCount := 0
	for _, issue := range issues {
		if issue.Severity == linter.SeverityFatal {
			fatalCount++
		} else if issue.Severity == linter.SeverityError {
			errorCount++
		} else if issue.Severity == linter.SeverityWarning {
			warningCount++
		}
	}

	if fatalCount > 0 {
		os.Exit(2)
	}

	if errorCount > 0 {
		os.Exit(1)
	}

	if failOnWarning && warningCount > 0 {
		os.Exit(4)
	}

	return nil
}
