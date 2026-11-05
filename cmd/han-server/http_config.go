package main

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

var errInvalidHTTPAddress = errors.New("invalid HTTP listen address")

func validateHTTPAddress(address string) error {
	if address == "" || strings.TrimSpace(address) != address {
		return errInvalidHTTPAddress
	}

	host, portText, err := net.SplitHostPort(address)
	if err != nil || host == "" || portText == "" {
		return errInvalidHTTPAddress
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return errInvalidHTTPAddress
	}
	if strings.Contains(host, ":") {
		if !strings.HasPrefix(address, "[") {
			return errInvalidHTTPAddress
		}
	} else if strings.HasPrefix(address, "[") || ip.To4() == nil {
		return errInvalidHTTPAddress
	}

	for i := range len(portText) {
		if portText[i] < '0' || portText[i] > '9' {
			return errInvalidHTTPAddress
		}
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errInvalidHTTPAddress
	}
	return nil
}
