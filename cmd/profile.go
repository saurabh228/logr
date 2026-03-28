package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/saurabh/logr/internal/filter"
	"github.com/saurabh/logr/internal/profile"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage saved filter profiles",
	Long:  `Save, load, list, and delete named filter profiles stored in ~/.logr/profiles/.`,
}

var profileSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save current flags as a named profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := buildConfigFromFlags()
		if err != nil {
			return err
		}
		if err := profile.Save(name, cfg); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Profile %q saved to ~/.logr/profiles/%s.toml\n", name, name)
		return nil
	},
}

var profileLoadCmd = &cobra.Command{
	Use:   "load <name>",
	Short: "Print the contents of a saved profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := profile.Load(name)
		if err != nil {
			return err
		}
		printConfig(cfg)
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved profiles",
	RunE: func(cmd *cobra.Command, _ []string) error {
		names, err := profile.List()
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Fprintln(os.Stdout, "No profiles saved yet.")
			return nil
		}
		for _, n := range names {
			fmt.Fprintln(os.Stdout, n)
		}
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a saved profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := profile.Delete(name); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Profile %q deleted.\n", name)
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileSaveCmd)
	profileCmd.AddCommand(profileLoadCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileDeleteCmd)
}

// buildConfigFromFlags constructs a filter.Config from the current global CLI flags.
func buildConfigFromFlags() (filter.Config, error) {
	cfg := filter.Config{
		MinLevel:      flagLevel,
		IncludeFields: flagIncludes,
		ExcludeFields: flagExcludes,
		HierPatterns:  flagHier,
		Services:      expandComma(flagServices),
	}
	if flagSuppressTTL != "" {
		d, err := time.ParseDuration(flagSuppressTTL)
		if err != nil {
			return cfg, fmt.Errorf("invalid --suppress-ttl value %q: %w", flagSuppressTTL, err)
		}
		cfg.SuppressTTL = d
	}
	return cfg, nil
}

// printConfig writes a human-readable summary of a filter.Config.
func printConfig(cfg filter.Config) {
	fmt.Fprintf(os.Stdout, "min_level:      %s\n", orNone(cfg.MinLevel))
	fmt.Fprintf(os.Stdout, "services:       %s\n", orNone(strings.Join(cfg.Services, ", ")))
	fmt.Fprintf(os.Stdout, "include_fields: %s\n", orNone(strings.Join(cfg.IncludeFields, ", ")))
	fmt.Fprintf(os.Stdout, "exclude_fields: %s\n", orNone(strings.Join(cfg.ExcludeFields, ", ")))
	fmt.Fprintf(os.Stdout, "hier_patterns:  %s\n", orNone(strings.Join(cfg.HierPatterns, ", ")))
	if cfg.SuppressTTL > 0 {
		fmt.Fprintf(os.Stdout, "suppress_ttl:   %s\n", cfg.SuppressTTL)
	} else {
		fmt.Fprintf(os.Stdout, "suppress_ttl:   (disabled)\n")
	}
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
