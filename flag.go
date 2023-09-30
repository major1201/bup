package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var (
	helpFlag           = pflag.BoolP("help", "h", false, "show help")
	bufsizeFlag        = pflag.Int("buf", 40, "input buffer size & pipeline buffer sizes in `megabytes` (MiB)")
	browserBufsizeFlag = pflag.Int("bbuf", 1, "browser buffer size & pipeline buffer sizes in `megabytes` (MiB)")
	shellFlag          = pflag.StringArrayP("exec", "e", nil, "`command` to run pipeline with; repeat multiple times to pass multi-word command; defaults to '-e=$SHELL -e=-c'")
)

func init() {
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:

    bup - Ultimate Plumber for browser

    redirect your command result to browser interactively, with instant live preview of command results, idea from [up](https://github.com/akavel/up)

    $ lshw |& bup

Options:
`, os.Args[0])
		pflag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nHome page: https://github.com/major1201/bup")
	}
}
