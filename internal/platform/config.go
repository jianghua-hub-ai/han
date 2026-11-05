// Package platform provides shared transport, configuration, and observability contracts.
package platform

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	// EnvProcessingRegionRef identifies the deployed processing region record.
	EnvProcessingRegionRef = "HAN_PROCESSING_REGION_REF"
	// EnvProcessingControllerRef identifies the deployed data controller record.
	EnvProcessingControllerRef = "HAN_PROCESSING_CONTROLLER_REF"
	// EnvProcessingLegalBasisRef identifies the deployed processing-basis record.
	EnvProcessingLegalBasisRef = "HAN_PROCESSING_LEGAL_BASIS_REF"
	// EnvSMTPProcessorRef identifies the approved SMTP processor record.
	EnvSMTPProcessorRef = "HAN_SMTP_PROCESSOR_REF"
	// EnvPublicOrigin identifies the canonical browser origin.
	EnvPublicOrigin = "HAN_PUBLIC_ORIGIN"
	// EnvTrustedProxyCIDRs identifies the trusted direct-proxy networks.
	EnvTrustedProxyCIDRs = "HAN_TRUSTED_PROXY_CIDRS"
	// EnvSMTPTLSMode identifies the required SMTP TLS mode.
	EnvSMTPTLSMode = "HAN_SMTP_TLS_MODE"
	// EnvSMTPHost identifies the SMTP server host.
	EnvSMTPHost = "HAN_SMTP_HOST"
	// EnvSMTPServerName identifies the SMTP certificate server name.
	EnvSMTPServerName = "HAN_SMTP_SERVER_NAME"
	// EnvSMTPCAPath identifies an optional controlled CA bundle path.
	EnvSMTPCAPath = "HAN_SMTP_CA_BUNDLE"
)

// EnvironmentLookup reads one environment variable.
type EnvironmentLookup func(string) (string, bool)

// ProcessingGateConfig contains deployment references required by ProcessingGate.
type ProcessingGateConfig struct {
	RegionRef        string
	ControllerRef    string
	LegalBasisRef    string
	SMTPProcessorRef string
}

// DeploymentConfig contains validated non-secret deployment security inputs.
type DeploymentConfig struct {
	ProcessingGate    ProcessingGateConfig
	PublicOrigin      string
	TrustedProxyCIDRs []netip.Prefix
	SMTPTLSMode       string
	SMTPHost          string
	SMTPServerName    string
	SMTPCAPath        string
}

// LoadProcessingGateConfig reads and validates ProcessingGate environment values.
func LoadProcessingGateConfig(lookup EnvironmentLookup) (ProcessingGateConfig, error) {
	if lookup == nil {
		return ProcessingGateConfig{}, fmt.Errorf("environment lookup is required")
	}

	regionRef, err := requiredEnvironmentValue(lookup, EnvProcessingRegionRef)
	if err != nil {
		return ProcessingGateConfig{}, err
	}
	controllerRef, err := requiredEnvironmentValue(lookup, EnvProcessingControllerRef)
	if err != nil {
		return ProcessingGateConfig{}, err
	}
	legalBasisRef, err := requiredEnvironmentValue(lookup, EnvProcessingLegalBasisRef)
	if err != nil {
		return ProcessingGateConfig{}, err
	}
	smtpProcessorRef, err := requiredEnvironmentValue(lookup, EnvSMTPProcessorRef)
	if err != nil {
		return ProcessingGateConfig{}, err
	}

	config := ProcessingGateConfig{
		RegionRef:        regionRef,
		ControllerRef:    controllerRef,
		LegalBasisRef:    legalBasisRef,
		SMTPProcessorRef: smtpProcessorRef,
	}
	if err := config.Validate(); err != nil {
		return ProcessingGateConfig{}, err
	}
	return config, nil
}

// LoadProcessingGateConfigFromEnvironment reads ProcessingGate configuration from the process environment.
func LoadProcessingGateConfigFromEnvironment() (ProcessingGateConfig, error) {
	return LoadProcessingGateConfig(os.LookupEnv)
}

// LoadDeploymentConfig reads and validates deployment security environment values.
func LoadDeploymentConfig(lookup EnvironmentLookup) (DeploymentConfig, error) {
	processingGate, err := LoadProcessingGateConfig(lookup)
	if err != nil {
		return DeploymentConfig{}, err
	}

	originValue, err := requiredEnvironmentValue(lookup, EnvPublicOrigin)
	if err != nil {
		return DeploymentConfig{}, err
	}
	publicOrigin, err := normalizePublicOrigin(originValue)
	if err != nil {
		return DeploymentConfig{}, fmt.Errorf("environment variable %s must be a canonical HTTP(S) origin", EnvPublicOrigin)
	}

	proxyValue, err := requiredEnvironmentValue(lookup, EnvTrustedProxyCIDRs)
	if err != nil {
		return DeploymentConfig{}, err
	}
	trustedProxyCIDRs, err := parseTrustedProxyCIDRs(proxyValue)
	if err != nil {
		return DeploymentConfig{}, err
	}

	smtpTLSMode, err := requiredEnvironmentValue(lookup, EnvSMTPTLSMode)
	if err != nil {
		return DeploymentConfig{}, err
	}
	if smtpTLSMode != "implicit_tls" && smtpTLSMode != "starttls_required" {
		return DeploymentConfig{}, fmt.Errorf("environment variable %s has an unsupported value", EnvSMTPTLSMode)
	}
	smtpHost, err := requiredEnvironmentValue(lookup, EnvSMTPHost)
	if err != nil {
		return DeploymentConfig{}, err
	}
	smtpServerName, err := requiredEnvironmentValue(lookup, EnvSMTPServerName)
	if err != nil {
		return DeploymentConfig{}, err
	}

	return DeploymentConfig{
		ProcessingGate:    processingGate,
		PublicOrigin:      publicOrigin,
		TrustedProxyCIDRs: trustedProxyCIDRs,
		SMTPTLSMode:       smtpTLSMode,
		SMTPHost:          smtpHost,
		SMTPServerName:    smtpServerName,
		SMTPCAPath:        optionalEnvironmentValue(lookup, EnvSMTPCAPath),
	}, nil
}

// LoadDeploymentConfigFromEnvironment reads deployment security configuration from the process environment.
func LoadDeploymentConfigFromEnvironment() (DeploymentConfig, error) {
	return LoadDeploymentConfig(os.LookupEnv)
}

// Validate rejects incomplete ProcessingGate configuration.
func (c ProcessingGateConfig) Validate() error {
	required := []struct {
		key   string
		value string
	}{
		{key: EnvProcessingRegionRef, value: c.RegionRef},
		{key: EnvProcessingControllerRef, value: c.ControllerRef},
		{key: EnvProcessingLegalBasisRef, value: c.LegalBasisRef},
		{key: EnvSMTPProcessorRef, value: c.SMTPProcessorRef},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("required environment variable %s is not set", item.key)
		}
	}
	return nil
}

func requiredEnvironmentValue(lookup EnvironmentLookup, key string) (string, error) {
	value, exists := lookup(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

func optionalEnvironmentValue(lookup EnvironmentLookup, key string) string {
	value, _ := lookup(key)
	return strings.TrimSpace(value)
}

func normalizePublicOrigin(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported scheme")
	}
	if parsed.Host == "" || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.User != nil {
		return "", fmt.Errorf("authority is invalid")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" {
		return "", fmt.Errorf("query and fragment are not allowed")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("path is not allowed")
	}

	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	hasExplicitPort := false
	if strings.HasPrefix(parsed.Host, "[") {
		closingBracket := strings.LastIndex(parsed.Host, "]")
		if closingBracket < 0 {
			return "", fmt.Errorf("authority is invalid")
		}
		portSuffix := parsed.Host[closingBracket+1:]
		if portSuffix != "" {
			if portSuffix == ":" || !strings.HasPrefix(portSuffix, ":") {
				return "", fmt.Errorf("port is invalid")
			}
			hasExplicitPort = true
		}
	} else if strings.Contains(parsed.Host, ":") {
		if strings.HasSuffix(parsed.Host, ":") {
			return "", fmt.Errorf("port is invalid")
		}
		hasExplicitPort = true
	}

	if hasExplicitPort {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return "", fmt.Errorf("port is invalid")
		}
		port = strconv.Itoa(portNumber)
		if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
			port = ""
		}
	}

	normalizedHost := host
	if strings.Contains(host, ":") || port != "" {
		normalizedHost = net.JoinHostPort(host, port)
		if port == "" {
			normalizedHost = strings.TrimSuffix(normalizedHost, ":")
		}
	}
	return (&url.URL{
		Scheme: scheme,
		Host:   normalizedHost,
	}).String(), nil
}

func parseTrustedProxyCIDRs(value string) ([]netip.Prefix, error) {
	parts := strings.Split(value, ",")
	prefixes := make([]netip.Prefix, 0, len(parts))
	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("environment variable %s contains an empty CIDR at position %d", EnvTrustedProxyCIDRs, index+1)
		}
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			return nil, fmt.Errorf("environment variable %s contains an invalid CIDR at position %d", EnvTrustedProxyCIDRs, index+1)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}
