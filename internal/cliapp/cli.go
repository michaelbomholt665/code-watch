package cliapp

import "flag"

const versionString = "1.0.0"
const defaultConfigPath = "./circular.toml"

type cliOptions struct {
	configPath string
	once       bool
	ui         bool
	trace      bool
	impact     string
	verbose    bool
	version    bool
	args       []string
}

func parseOptions(args []string) (cliOptions, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("circular", flag.ContinueOnError)

	fs.StringVar(&opts.configPath, "config", defaultConfigPath, "Path to config file")
	fs.BoolVar(&opts.once, "once", false, "Run single scan and exit")
	fs.BoolVar(&opts.ui, "ui", false, "Enable terminal UI mode")
	fs.BoolVar(&opts.trace, "trace", false, "Trace shortest import chain between two modules")
	fs.StringVar(&opts.impact, "impact", "", "Analyze change impact for a file path or module")
	fs.BoolVar(&opts.verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&opts.version, "version", false, "Print version and exit")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}

	opts.args = fs.Args()
	return opts, nil
}
