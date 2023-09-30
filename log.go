package main

import (
	"flag"
	"log"

	"github.com/go-logr/glogr"
	"github.com/go-logr/logr"
)

var (
	L   logr.Logger
	Std *log.Logger
)

func init() {
	_ = flag.Set("v", "5")
	_ = flag.Set("logtostderr", "true")
	// flag.Parse()
	L = glogr.NewWithOptions(glogr.Options{})
	Std = NewStd(L)
}

func NewStd(l logr.Logger) *log.Logger {
	return log.New(&writer{l: l}, "", log.Lshortfile)
}

type writer struct {
	l logr.Logger
}

func (w *writer) Write(p []byte) (n int, err error) {
	w.l.Info(string(p))
	return len(p), nil
}
