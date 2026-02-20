package cli

import (
	"circular/internal/shared/version"
	"flag"
)

const versionString = version.Version
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
	reportMarkdown bool
	sarif          string
	scanHistory    int
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
	fs.BoolVar(&opts.reportMarkdown, "report-md", false, "Generate markdown analysis report output (uses output.markdown or analysis-report.md)")
	fs.BoolVar(&opts.verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&opts.version, "version", false, "Print version and exit")
	fs.StringVar(&opts.sarif, "sarif", "", "Write SARIF v2.1.0 report to this path (for GitHub Code Scanning)")
	fs.IntVar(&opts.scanHistory, "scan-history", 0, "Scan last N git commits for deleted secrets (0 = disabled)")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}

	opts.args = fs.Args()
	return opts, nil
}
