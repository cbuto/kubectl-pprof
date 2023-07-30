package portforward

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/phayes/freeport"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwarderOption func(forwarder *PortForwarder)

type PortForwarder struct {
	host       string
	restConfig *rest.Config
	out        io.Writer
	errOut     io.Writer
}

func NewPortForward(options ...PortForwarderOption) (*PortForwarder, error) {
	portForwarder := &PortForwarder{}
	for _, opt := range options {
		opt(portForwarder)
	}

	if portForwarder.restConfig == nil {
		return nil, fmt.Errorf("RESTConfig cannot be nil")
	}

	portForwarder.host = portForwarder.restConfig.Host

	return portForwarder, nil
}

func WithRESTConfig(restConfig *rest.Config) PortForwarderOption {
	return func(forwarder *PortForwarder) {
		forwarder.restConfig = restConfig
	}
}

func WithOutput(out io.Writer, errOut io.Writer) PortForwarderOption {
	return func(forwarder *PortForwarder) {
		forwarder.out = out
		forwarder.errOut = errOut
	}
}

func (f *PortForwarder) PortForward(
	ctx context.Context,
	namespace,
	pod string,
	port int,
) (int, chan struct{}, error) {
	var localPort int

	var readyCh chan struct{}

	stopCh, readyCh, errCh := make(chan struct{}, 1), make(chan struct{}, 1), make(chan error)

	roundTripper, upgrader, err := spdy.RoundTripperFor(f.restConfig)
	if err != nil {
		return localPort, nil, fmt.Errorf("failed to get round tripper: %w", err)
	}

	hostURL, err := url.Parse(f.restConfig.Host)
	if err != nil {
		return localPort, nil, fmt.Errorf("parsing host from RESTConfig failed: %w", err)
	}

	portForwardURL := &url.URL{
		Scheme: "https",
		Path: path.Join(
			hostURL.Path,
			"api/v1/namespaces",
			url.PathEscape(namespace),
			"pods",
			url.PathEscape(pod),
			"portforward",
		),
		Host: hostURL.Host,
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, portForwardURL)

	localPort, err = freeport.GetFreePort()
	if err != nil {
		return localPort, nil, fmt.Errorf("failed to find free port on host: %w", err)
	}

	ports := []string{fmt.Sprintf("%d:%d", localPort, port)}
	portforwarder, err := portforward.New(dialer, ports, stopCh, readyCh, f.out, f.errOut)

	if err != nil {
		return localPort, nil, fmt.Errorf("failed to get port-forwarder: %w", err)
	}

	go func() {
		if err := portforwarder.ForwardPorts(); err != nil {
			errCh <- err
		}

		close(errCh)
	}()

	select {
	case <-readyCh:
	case err = <-errCh:
		if err != nil {
			return localPort, nil, fmt.Errorf("port-forward failed: %w", err)
		}
	}

	return localPort, stopCh, nil
}
