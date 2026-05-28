package main

import (
	"flag"
	"fmt"
	"os"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/smoke"
)

func main() {
	var opts smoke.Options

	flag.StringVar(&opts.ReaderName, "reader", "", "reader name substring; empty uses the first available reader")
	flag.StringVar(&opts.AppletAID, "aid", smoke.DefaultAppletAID, "applet AID hex")
	flag.StringVar(&opts.PrivateKeyPath, "private-key", "", "ECDSA private key PEM path; default auto-detects the local development key")
	flag.IntVar(&opts.RecordCount, "records", smoke.DefaultRecordCount, "number of direct seclient records to exercise")
	flag.BoolVar(&opts.Debug, "debug", false, "enable mpc_core/seclient debug logs")
	flag.BoolVar(&opts.SkipDirect, "skip-direct", false, "skip direct mpc_core/seclient checks")
	flag.BoolVar(&opts.SkipService, "skip-service", false, "skip SecurityService checks")
	flag.Parse()

	if opts.Debug {
		_ = clog.Init(clog.Config{
			Level:         clog.DebugLevel,
			Format:        clog.FormatConsole,
			Filename:      "logs/se-smoke.log",
			Name:          "default",
			ConsoleOutput: true,
			EnableCaller:  false,
			EnableColor:   false,
		})
		defer clog.Sync()
	}

	opts.Output = os.Stdout
	if err := smoke.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "\n[FAIL] %v\n", err)
		os.Exit(1)
	}
}
