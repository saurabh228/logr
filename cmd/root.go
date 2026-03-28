package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/saurabh/logr/internal/filter"
	"github.com/saurabh/logr/internal/license"
	"github.com/saurabh/logr/internal/parser"
	"github.com/saurabh/logr/internal/profile"
	"github.com/saurabh/logr/internal/render"
	"github.com/saurabh/logr/internal/tail"
	"github.com/saurabh/logr/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Global flag values.
var (
	flagProfile     string
	flagLevel       string
	flagServices    []string
	flagIncludes    []string
	flagExcludes    []string
	flagHier        []string
	flagSuppressTTL string
	flagNoColor     bool
	flagJSON        bool
	flagLicenseKey  string
	flagSaveProfile string
	flagKeys         []string
	flagFollow       bool
	flagTUI          bool
	flagServiceField string
	flagHierField    string
)

// rootCmd is the top-level cobra command.  When invoked with no subcommand it
// reads from stdin (or a file path argument), applies the requested filters and
// renders the output.
var rootCmd = &cobra.Command{
	Use:   "logr [file]",
	Short: "Microservice-aware JSON log filter with saved profiles",
	Long: `logr — a fast, microservice-aware JSON log filter.

Pipe JSON log lines into logr or pass a file path as an argument.

Examples:
  kubectl logs -f my-pod | logr --level warn
  cat service.log | logr --service payment --hier "payment.**"
  logr service.log --level error --tui
  logr service.log --follow
  logr service.log --keys request_id,status_code
  cat service.log | logr --profile prod-errors
`,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runFilter,
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	pf := rootCmd.PersistentFlags()

	pf.StringVarP(&flagProfile, "profile", "p", "", "load a named filter profile")
	pf.StringVarP(&flagLevel, "level", "l", "", "minimum log level (debug|info|warn|error|fatal)")
	pf.StringSliceVarP(&flagServices, "service", "s", nil, "filter by service name (comma-separated or repeated)")
	pf.StringArrayVarP(&flagIncludes, "include", "i", nil, "include entries matching field=value (repeatable)")
	pf.StringArrayVarP(&flagExcludes, "exclude", "e", nil, "exclude entries matching field=value (repeatable)")
	pf.StringArrayVarP(&flagHier, "hier", "H", nil, "filter by hier path pattern, e.g. payment.** (repeatable)")
	pf.StringVar(&flagSuppressTTL, "suppress-ttl", "", "TTL for noise suppression, e.g. 30s or 1m (0 = disabled)")
	pf.BoolVar(&flagNoColor, "no-color", false, "disable colorized output")
	pf.BoolVar(&flagJSON, "json", false, "output entries as JSON instead of pretty-print")
	pf.StringVar(&flagLicenseKey, "license", "", "verify and cache a Gumroad license key")
	pf.StringVar(&flagSaveProfile, "save-profile", "", "save current flags as a named profile, then exit")
	pf.StringSliceVarP(&flagKeys, "keys", "k", nil, "show only these extra field keys, e.g. request_id,status_code")
	pf.BoolVarP(&flagFollow, "follow", "f", false, "follow file for new lines (requires a file argument, like tail -f)")
	pf.BoolVarP(&flagTUI, "tui", "T", false, "open interactive TUI viewer")
	pf.StringVar(&flagServiceField, "service-field", "", "JSON field to use as service name (e.g. name, logger, app_name)")
	pf.StringVar(&flagHierField, "hier-field", "", "JSON field to use as hier path (e.g. module, path, logger)")

	_ = viper.BindPFlag("license_key", pf.Lookup("license"))

	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err == nil {
		cfgDir := filepath.Join(home, ".logr")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(cfgDir)
		_ = viper.ReadInConfig()
	}
	viper.SetEnvPrefix("LOGR")
	viper.AutomaticEnv()
}

func runFilter(cmd *cobra.Command, args []string) error {
	// Handle --license flag (verify and cache, then exit).
	if flagLicenseKey != "" {
		if err := license.Verify(flagLicenseKey); err != nil {
			return fmt.Errorf("license verification failed: %w", err)
		}
		fmt.Fprintln(os.Stdout, "License verified and cached for 30 days.")
		return nil
	}

	// Resolve license key: CLI flag > viper config > cache.
	key := viper.GetString("license_key")
	if key == "" {
		key = license.KeyFromCache()
	}
	if err := license.Verify(key); err != nil {
		if os.Getenv("LOGR_DEV") != "1" {
			fmt.Fprintf(os.Stderr, "logr: %v\n", err)
			fmt.Fprintln(os.Stderr, "Run: logr --license <your-key>")
			os.Exit(1)
		}
	}

	// Build filter config from profile, then override with CLI flags.
	cfg := filter.Config{}
	if flagProfile != "" {
		loaded, err := profile.Load(flagProfile)
		if err != nil {
			return fmt.Errorf("cannot load profile %q: %w", flagProfile, err)
		}
		cfg = loaded
	}

	if flagLevel != "" {
		cfg.MinLevel = flagLevel
	}
	if len(flagServices) > 0 {
		cfg.Services = expandComma(flagServices)
	}
	if len(flagIncludes) > 0 {
		cfg.IncludeFields = flagIncludes
	}
	if len(flagExcludes) > 0 {
		cfg.ExcludeFields = flagExcludes
	}
	if len(flagHier) > 0 {
		cfg.HierPatterns = flagHier
	}
	if flagSuppressTTL != "" {
		d, err := time.ParseDuration(flagSuppressTTL)
		if err != nil {
			return fmt.Errorf("invalid --suppress-ttl value %q: %w", flagSuppressTTL, err)
		}
		cfg.SuppressTTL = d
	}
	if len(flagKeys) > 0 {
		cfg.Keys = expandComma(flagKeys)
	}

	// --save-profile writes config and exits before touching any input.
	if flagSaveProfile != "" {
		if err := profile.Save(flagSaveProfile, cfg); err != nil {
			return fmt.Errorf("cannot save profile: %w", err)
		}
		fmt.Fprintf(os.Stdout, "Profile %q saved.\n", flagSaveProfile)
		return nil
	}

	if flagNoColor {
		color.NoColor = true
	}

	opts := render.Options{
		NoColor: flagNoColor,
		JSON:    flagJSON,
		Keys:    cfg.Keys,
	}

	parseOpts := parser.Options{
		ServiceField: flagServiceField,
		HierField:    flagHierField,
	}

	// --follow requires a file argument.
	if flagFollow {
		if len(args) == 0 {
			return fmt.Errorf("--follow requires a file argument")
		}
		engine := filter.New(cfg)
		return tail.Follow(args[0], engine, opts, parseOpts, os.Stdout)
	}

	// Determine input source: file arg or stdin.
	var entries []parser.LogEntry
	readAll := flagTUI // TUI needs all entries in memory

	if len(args) == 1 {
		f, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("cannot open %q: %w", args[0], err)
		}
		defer f.Close()

		if readAll {
			entries = scanAll(f, parseOpts)
		} else {
			engine := filter.New(cfg)
			return scanAndRender(bufio.NewScanner(f), engine, opts, parseOpts)
		}
	} else {
		if readAll {
			entries = scanAll(os.Stdin, parseOpts)
		} else {
			engine := filter.New(cfg)
			return scanAndRender(bufio.NewScanner(os.Stdin), engine, opts, parseOpts)
		}
	}

	// TUI mode — entries already loaded.
	if flagTUI {
		return tui.Run(entries, cfg, opts)
	}

	return nil
}

// scanAndRender reads from scanner, filters, and writes to stdout.
func scanAndRender(scanner *bufio.Scanner, engine *filter.Engine, opts render.Options, parseOpts parser.Options) error {
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		entry := parser.ParseWith(scanner.Bytes(), parseOpts)
		if engine.Pass(entry) {
			render.Render(entry, os.Stdout, opts)
		}
	}
	return scanner.Err()
}

// scanAll reads all lines from r and returns the parsed entries.
func scanAll(r *os.File, parseOpts parser.Options) []parser.LogEntry {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var entries []parser.LogEntry
	for scanner.Scan() {
		entries = append(entries, parser.ParseWith(scanner.Bytes(), parseOpts))
	}
	return entries
}

// expandComma splits each element on commas and returns a flat slice.
func expandComma(ss []string) []string {
	var out []string
	for _, s := range ss {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
