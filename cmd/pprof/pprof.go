package pprof

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"runtime/pprof"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/rest"

	"github.com/cbuto/kubectl-pprof/internal/portforward"
	"github.com/cbuto/kubectl-pprof/internal/pprofgetter"
)

var pprofExample = `
	# collect a heap profile a pod
	%[1]s pprof <pod name> --profile heap --seconds 10
	
	# collect a cpu profile and output profile to /tmp/
	%[1]s pprof <pod name> --port 8080 -n <namespace> --profile cpu --output /tmp/ --seconds 30

	# pass a profile directly to go tool pprof (suppress output with -q)
	%[1]s pprof <pod> --port 8080 -n <namespace> --profile cpu -q | xargs go tool pprof -http=:8080

`

const (
	DefaultProfileSeconds = 10
	DefaultPProfPort      = 8080
)

type pprofCmdOptions struct {
	genericiooptions.IOStreams
	restConfig  *rest.Config
	configFlags *genericclioptions.ConfigFlags
	profile     string
	seconds     int
	port        int
	outDir      string
	namespace   string
	pod         string
	quiet       bool
}

func newPProfOptions(streams genericiooptions.IOStreams) *pprofCmdOptions {
	return &pprofCmdOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewPProfCmd(streams genericiooptions.IOStreams) *cobra.Command {
	opts := newPProfOptions(streams)

	cmd := &cobra.Command{
		Use:          "pprof [pod] [flags]",
		Short:        "Collects the specified pprof profile from a pod",
		Example:      fmt.Sprintf(pprofExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := opts.Complete(c, args); err != nil {
				return err
			}
			if err := opts.Validate(); err != nil {
				return err
			}
			if err := opts.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "suppresses output and only prints the output file")
	cmd.Flags().StringVar(&opts.profile,
		"profile",
		"cpu", "type of profile to collect (heap, cpu, block, goroutine, mutex, or threadcreate)")
	cmd.Flags().StringVar(&opts.outDir, "output", "./", "path to a directory to write the profile")
	cmd.Flags().IntVar(&opts.seconds, "seconds", DefaultProfileSeconds, "amount of seconds to collect the profile")
	cmd.Flags().IntVar(&opts.port, "port", DefaultPProfPort, "pprof port")

	opts.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *pprofCmdOptions) Complete(cmd *cobra.Command, args []string) error {
	var err error
	o.namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()

	if err != nil {
		return fmt.Errorf("failed to get namespace from config: %w", err)
	}

	if len(args) != 1 {
		return fmt.Errorf("invalid number of arguments, use --help to see example usage")
	}

	o.pod = args[0]
	o.restConfig, err = o.configFlags.ToRESTConfig()

	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	return nil
}

func (o *pprofCmdOptions) Validate() error {
	if o.profile == "" {
		return fmt.Errorf("must select type of profile to collect")
	}

	if o.profile != "cpu" {
		profile := pprof.Lookup(o.profile)
		if profile == nil {
			return fmt.Errorf("unknown profile: %s", o.profile)
		}
	}

	fileInfo, err := os.Stat(o.outDir)

	if err != nil {
		return fmt.Errorf("failed to check if --output is a dir: %w", err)
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("--output flag must be set to a directory: %s", o.outDir)
	}

	return nil
}

func (o *pprofCmdOptions) Run() error {
	var err error

	ctx := context.TODO()

	out, errOut := o.IOStreams.Out, o.IOStreams.ErrOut
	if o.quiet {
		out, errOut = nil, nil
	}

	portforwarder, err := portforward.NewPortForward(
		portforward.WithRESTConfig(o.restConfig),
		portforward.WithOutput(out, errOut))
	if err != nil {
		return fmt.Errorf("failed to get port-forwarder: %w", err)
	}

	localPort, stopCh, err := portforwarder.PortForward(ctx, o.namespace, o.pod, o.port)

	if err != nil {
		return fmt.Errorf("port-forwarding failed: %w", err)
	}

	defer close(stopCh)

	if !o.quiet {
		_, err = o.IOStreams.Out.Write([]byte(fmt.Sprintf("collecting profile from %s (for %d secs)...\n", o.pod, o.seconds)))
		if err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	outFilePath := path.Join(o.outDir, fmt.Sprintf("%s_%s_%d.out", o.pod, o.profile, time.Now().Unix()))
	pprofHostURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", localPort),
	}

	getter, err := pprofgetter.NewPProfGetter(pprofgetter.WithHostURL(pprofHostURL))
	if err != nil {
		return fmt.Errorf("failed to build pprof getter: %w", err)
	}

	err = getter.Get(ctx, o.profile, o.seconds, outFilePath)
	if err != nil {
		return fmt.Errorf("failed to collect profile: %w", err)
	}

	_, err = o.IOStreams.Out.Write([]byte(fmt.Sprintf("%s\n", outFilePath)))
	if err != nil {
		return fmt.Errorf("failed to output profile filename: %w", err)
	}

	return nil
}
