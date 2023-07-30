package pprofgetter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPProfGetter(t *testing.T) {
	fakeURL := &url.URL{
		Host: "foo",
	}
	tests := []struct {
		name    string
		options []Option
		want    *PProfGetter
		wantErr bool
	}{
		{
			name:    "no host url",
			options: nil,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "with host url",
			options: []Option{WithHostURL(fakeURL)},
			want:    &PProfGetter{hostURL: fakeURL},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPProfGetter(tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPProfGetter() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPProfGetter() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPProfGetter_Get(t *testing.T) {
	expectedOutput := "foo"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(expectedOutput))
		assert.NoError(t, err)
	}))

	defer testServer.Close()

	parsedURL, err := url.Parse(testServer.URL)
	assert.NoError(t, err)

	fakeURL := &url.URL{
		Scheme: "http",
		Host:   parsedURL.Host,
	}

	outputFileDir, err := os.MkdirTemp("", "pproftest")
	assert.NoError(t, err)

	defer os.RemoveAll(outputFileDir)

	outFilePath := path.Join(outputFileDir, "foo.out")

	p := &PProfGetter{
		hostURL: fakeURL,
	}

	err = p.Get(context.TODO(), CPUProfileName, 0, outFilePath)
	assert.NoError(t, err)

	actual, err := os.ReadFile(outFilePath)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, string(actual))
}

func TestPProfGetter_GetFailed(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	parsedURL, err := url.Parse(testServer.URL)
	assert.NoError(t, err)
	fakeURL := &url.URL{
		Scheme: "http",
		Host:   parsedURL.Host,
	}

	outputFileDir, err := os.MkdirTemp("", "pproftest")
	assert.NoError(t, err)

	defer os.RemoveAll(outputFileDir)

	outFilePath := path.Join(outputFileDir, "foo.out")

	p := &PProfGetter{
		hostURL: fakeURL,
	}
	err = p.Get(context.TODO(), CPUProfileName, 0, outFilePath)
	assert.ErrorContains(t, err, ErrPProfEndpointFailed.Error())
}
