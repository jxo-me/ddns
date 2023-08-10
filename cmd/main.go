package main

import (
	"flag"
	"fmt"
	"github.com/judwhite/go-svc"
	"log"
	"os"
	"runtime"
)

var (
	cfgFile      string
	services     stringList
	outputFormat string
)

func init() {
	var printVersion bool
	flag.StringVar(&cfgFile, "C", "", "configuration file")
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.Parse()
	if printVersion {
		fmt.Fprintf(os.Stdout, "ddns %s (%s %s/%s)\n",
			version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}
}

func main() {
	p := &program{}
	if err := svc.Run(p); err != nil {
		log.Fatal(err)
	}
}
