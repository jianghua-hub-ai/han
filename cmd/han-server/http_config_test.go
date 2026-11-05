package main

import "testing"

func TestValidateHTTPAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{name: "IPv4 minimum port", address: "127.0.0.1:1"},
		{name: "IPv4 maximum port", address: "0.0.0.0:65535"},
		{name: "IPv6 minimum port", address: "[::1]:1"},
		{name: "IPv6 maximum port", address: "[2001:db8::1]:65535"},
		{name: "empty", wantErr: true},
		{name: "whitespace", address: " \t ", wantErr: true},
		{name: "empty host", address: ":8080", wantErr: true},
		{name: "domain", address: "localhost:8080", wantErr: true},
		{name: "port zero", address: "127.0.0.1:0", wantErr: true},
		{name: "port too high", address: "127.0.0.1:65536", wantErr: true},
		{name: "negative port", address: "127.0.0.1:-1", wantErr: true},
		{name: "service port", address: "127.0.0.1:http", wantErr: true},
		{name: "scheme", address: "http://127.0.0.1:8080", wantErr: true},
		{name: "path", address: "127.0.0.1:8080/path", wantErr: true},
		{name: "multiple addresses", address: "127.0.0.1:8080,127.0.0.2:8080", wantErr: true},
		{name: "missing port", address: "127.0.0.1", wantErr: true},
		{name: "unbracketed IPv6", address: "::1:8080", wantErr: true},
		{name: "IPv6 zone", address: "[fe80::1%eth0]:8080", wantErr: true},
		{name: "surrounding whitespace", address: " 127.0.0.1:8080 ", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateHTTPAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHTTPAddress(%q) error = %v, wantErr %v", tt.address, err, tt.wantErr)
			}
		})
	}
}
