package portforward

import (
	"io"
	"reflect"
	"testing"

	"k8s.io/client-go/rest"
)

func TestNewPortForward(t *testing.T) {
	host := "foo"
	fakeRestConfig := &rest.Config{
		Host: host,
	}
	fakeOutput := io.Discard
	tests := []struct {
		name    string
		options []PortForwarderOption
		want    *PortForwarder
		wantErr bool
	}{
		{
			name:    "no restconfig",
			options: []PortForwarderOption{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "with restconfig",
			options: []PortForwarderOption{WithRESTConfig(fakeRestConfig)},
			want:    &PortForwarder{host: host, restConfig: fakeRestConfig},
			wantErr: false,
		},
		{
			name:    "with output",
			options: []PortForwarderOption{WithRESTConfig(fakeRestConfig), WithOutput(fakeOutput, fakeOutput)},
			want:    &PortForwarder{host: host, restConfig: fakeRestConfig, errOut: fakeOutput, out: fakeOutput},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPortForward(tt.options...)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewPortForward() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPortForward() got = %v, want %v", got, tt.want)
			}
		})
	}
}
