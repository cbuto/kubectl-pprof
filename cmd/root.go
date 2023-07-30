package main

import (
	"os"

	"github.com/cbuto/kubectl-pprof/cmd/pprof"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-pprof", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := pprof.NewPProfCmd(genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
