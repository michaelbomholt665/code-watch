package cli

import "flag"

const versionString = "1.0.0"
const defaultConfigPath = "./data/config/circular.toml"

type cliOptions struct {
	configPath     string
	once           bool
	ui             bool
	trace          bool
	impact         string
	history        bool
	since          string
	historyWindow  string
	historyTSV     string
	historyJSON    string
	queryModules   bool
	queryFilter    string
	queryModule    string
	queryTrace     string
	queryTrends    bool
	queryLimit     int
	includeTests   bool
	verifyGrammars bool
	verbose        bool
	version        bool
	args           []string
}

func parseOptions(args []string) (cliOptions, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("circular", flag.ContinueOnError)

	fs.StringVar(&opts.configPath, "config", defaultConfigPath, "Path to config file")
	fs.BoolVar(&opts.once, "once", false, "Run single scan and exit")
	fs.BoolVar(&opts.ui, "ui", false, "Enable terminal UI mode")
	fs.BoolVar(&opts.trace, "trace", false, "Trace shortest import chain between two modules")
	fs.StringVar(&opts.impact, "impact", "", "Analyze change impact for a file path or module")
	fs.BoolVar(&opts.history, "history", false, "Enable local history snapshots and trend reporting")
	fs.StringVar(&opts.since, "since", "", "Include historical snapshots at/after this timestamp (RFC3339 or YYYY-MM-DD)")
	fs.StringVar(&opts.historyWindow, "history-window", "24h", "Moving-window duration for trend summaries (requires --history)")
	fs.StringVar(&opts.historyTSV, "history-tsv", "", "Write trend report TSV to this path (requires --history)")
	fs.StringVar(&opts.historyJSON, "history-json", "", "Write trend report JSON to this path (requires --history)")
	fs.BoolVar(&opts.queryModules, "query-modules", false, "List modules from shared query service")
	fs.StringVar(&opts.queryFilter, "query-filter", "", "Optional substring filter for --query-modules")
	fs.StringVar(&opts.queryModule, "query-module", "", "Print module details from shared query service")
	fs.StringVar(&opts.queryTrace, "query-trace", "", "Print dependency trace from shared query service (<from>:<to>)")
	fs.BoolVar(&opts.queryTrends, "query-trends", false, "Print historical trend slice from shared query service (requires --history)")
	fs.IntVar(&opts.queryLimit, "query-limit", 0, "Optional limit/depth control for query modes")
	fs.BoolVar(&opts.includeTests, "include-tests", false, "Include test files in analysis (Go: _test.go, Python: test_*.py)")
	fs.BoolVar(&opts.verifyGrammars, "verify-grammars", false, "Verify grammar artifacts against grammars/manifest.toml and exit")
	fs.BoolVar(&opts.verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&opts.version, "version", false, "Print version and exit")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}

	opts.args = fs.Args()
	return opts, nil
}
