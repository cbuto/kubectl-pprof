package pprofgetter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const CPUProfileName = "cpu"

var ErrPProfEndpointFailed = errors.New("pprof endpoint failed")

type PProfGetter struct {
	hostURL *url.URL
}

type Option func(getter *PProfGetter)

func WithHostURL(hostURL *url.URL) Option {
	return func(getter *PProfGetter) {
		getter.hostURL = hostURL
	}
}

func NewPProfGetter(options ...Option) (*PProfGetter, error) {
	pprofGetter := &PProfGetter{}
	for _, opt := range options {
		opt(pprofGetter)
	}

	if pprofGetter.hostURL == nil {
		return nil, fmt.Errorf("url cannot be nil")
	}

	return pprofGetter, nil
}

func (p *PProfGetter) Get(ctx context.Context, profile string, seconds int, outFilePath string) error {
	var profilePath string

	profilePath = fmt.Sprintf("/debug/pprof/%s", profile)
	if profile == CPUProfileName {
		profilePath = "/debug/pprof/profile"
	}

	p.hostURL.RawQuery = fmt.Sprintf("seconds=%d", seconds)
	fullURL := p.hostURL.JoinPath(profilePath)

	var body io.Reader
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL.String(), body)

	if err != nil {
		return fmt.Errorf("failed to build profile HTTP request %s: %w", fullURL.String(), err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("profile endpoint failed %s: %w", fullURL.String(), err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned status code %d: %w", fullURL.String(), resp.StatusCode, ErrPProfEndpointFailed)
	}

	file, err := os.Create(outFilePath)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %w", outFilePath, err)
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("unable to write profile to file %s: %w", outFilePath, err)
	}

	return nil
}
