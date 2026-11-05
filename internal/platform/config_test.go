package platform

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadProcessingGateConfig_MissingOrBlankValue_FailsClosed(t *testing.T) {
	valid := map[string]string{
		EnvProcessingRegionRef:     "region-record",
		EnvProcessingControllerRef: "controller-record",
		EnvProcessingLegalBasisRef: "basis-record",
		EnvSMTPProcessorRef:        "smtp-record",
	}

	tests := []struct {
		name  string
		key   string
		value string
		set   bool
	}{
		{name: "missing region", key: EnvProcessingRegionRef},
		{name: "blank controller", key: EnvProcessingControllerRef, value: " \t ", set: true},
		{name: "missing legal basis", key: EnvProcessingLegalBasisRef},
		{name: "missing SMTP processor", key: EnvSMTPProcessorRef},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := mapsClone(valid)
			if test.set {
				values[test.key] = test.value
			} else {
				delete(values, test.key)
			}

			_, err := LoadProcessingGateConfig(mapLookup(values))
			if err == nil {
				t.Fatal("LoadProcessingGateConfig() error = nil, want fail-closed error")
			}
			if !strings.Contains(err.Error(), test.key) {
				t.Fatalf("LoadProcessingGateConfig() error = %q, want missing key %q", err, test.key)
			}
			for _, value := range valid {
				if strings.Contains(err.Error(), value) {
					t.Fatalf("LoadProcessingGateConfig() error leaked configured value %q", value)
				}
			}
		})
	}
}

func TestLoadProcessingGateConfig_CompleteValues_ReturnsTrimmedConfig(t *testing.T) {
	values := map[string]string{
		EnvProcessingRegionRef:     " region-record ",
		EnvProcessingControllerRef: " controller-record ",
		EnvProcessingLegalBasisRef: " basis-record ",
		EnvSMTPProcessorRef:        " smtp-record ",
	}

	config, err := LoadProcessingGateConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadProcessingGateConfig() error = %v", err)
	}

	want := ProcessingGateConfig{
		RegionRef:        "region-record",
		ControllerRef:    "controller-record",
		LegalBasisRef:    "basis-record",
		SMTPProcessorRef: "smtp-record",
	}
	if config != want {
		t.Fatalf("LoadProcessingGateConfig() = %#v, want %#v", config, want)
	}
}

func mapsClone(source map[string]string) map[string]string {
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func mapLookup(values map[string]string) EnvironmentLookup {
	return func(key string) (string, bool) {
		value, exists := values[key]
		return value, exists
	}
}

func TestProcessingGateReferences_MissingOrBlank_FailClosedWithoutValueLeak(t *testing.T) {
	valid := map[string]string{
		EnvProcessingRegionRef:     "private-region-reference",
		EnvProcessingControllerRef: "private-controller-reference",
		EnvProcessingLegalBasisRef: "private-basis-reference",
		EnvSMTPProcessorRef:        "private-smtp-reference",
	}

	for key := range valid {
		for _, test := range []struct {
			name  string
			value string
			set   bool
		}{
			{name: "missing"},
			{name: "blank", value: " \t ", set: true},
		} {
			t.Run(key+"/"+test.name, func(t *testing.T) {
				values := mapsClone(valid)
				if test.set {
					values[key] = test.value
				} else {
					delete(values, key)
				}

				_, err := LoadProcessingGateConfig(mapLookup(values))
				if err == nil {
					t.Fatal("LoadProcessingGateConfig() error = nil, want fail-closed error")
				}
				if !strings.Contains(err.Error(), key) {
					t.Fatalf("LoadProcessingGateConfig() error = %q, want missing key %q", err, key)
				}
				for _, value := range valid {
					if strings.Contains(err.Error(), value) {
						t.Fatalf("LoadProcessingGateConfig() error leaked configured value %q", value)
					}
				}
			})
		}
	}
}

func TestProcessingGateArtifacts_ApprovedKeySet_IsExactAndEmpty(t *testing.T) {
	values := readDeploymentEnvironmentExample(t)
	if len(values) != len(deploymentEnvironmentKeys()) {
		t.Fatalf("environment example has %d assignments, want %d", len(values), len(deploymentEnvironmentKeys()))
	}
	for _, key := range deploymentEnvironmentKeys() {
		requireEmptyDeploymentValue(t, values, key)
	}
}

func TestProcessingGateArtifacts_UnsafeAssignments_AreRejected(t *testing.T) {
	valid := deploymentEnvironmentFixture()
	tests := []struct {
		name           string
		content        string
		sensitiveValue string
	}{
		{
			name:    "unknown key",
			content: valid + "HAN_UNKNOWN=\n",
		},
		{
			name:           "extra nonempty assignment",
			content:        valid + "HAN_EXTRA=private-extra-value\n",
			sensitiveValue: "private-extra-value",
		},
		{
			name:           "credential assignment",
			content:        valid + "HAN_SMTP_PASSWORD=private-password\n",
			sensitiveValue: "private-password",
		},
		{
			name:           "production endpoint",
			content:        strings.Replace(valid, "HAN_PUBLIC_ORIGIN=\n", "HAN_PUBLIC_ORIGIN=https://production.example\n", 1),
			sensitiveValue: "https://production.example",
		},
		{
			name:    "duplicate key",
			content: valid + "HAN_PUBLIC_ORIGIN=\n",
		},
		{
			name:    "missing approved key",
			content: strings.Replace(valid, "HAN_SMTP_CA_BUNDLE=\n", "", 1),
		},
	}
	for _, key := range deploymentEnvironmentKeys() {
		const sensitiveValue = "private-nonempty-value"
		tests = append(tests, struct {
			name           string
			content        string
			sensitiveValue string
		}{
			name:           "nonempty approved key " + key,
			content:        strings.Replace(valid, key+"=\n", key+"="+sensitiveValue+"\n", 1),
			sensitiveValue: sensitiveValue,
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseDeploymentEnvironmentExample(test.content)
			if err == nil {
				t.Fatal("parseDeploymentEnvironmentExample() error = nil, want unsafe assignment rejected")
			}
			if test.sensitiveValue != "" && strings.Contains(err.Error(), test.sensitiveValue) {
				t.Fatal("parseDeploymentEnvironmentExample() error leaked configured value")
			}
		})
	}
}

func TestOriginArtifacts_ApprovedKey_IsExplicitlyEmpty(t *testing.T) {
	values := readDeploymentEnvironmentExample(t)
	requireEmptyDeploymentValue(t, values, "HAN_PUBLIC_ORIGIN")
}

func TestTrustedProxyArtifacts_ApprovedKey_IsExplicitlyEmpty(t *testing.T) {
	values := readDeploymentEnvironmentExample(t)
	requireEmptyDeploymentValue(t, values, "HAN_TRUSTED_PROXY_CIDRS")
}

func readDeploymentEnvironmentExample(t *testing.T) map[string]string {
	t.Helper()

	content := readDeploymentArtifact(t, "deploy", "processing-gate.env.example")
	values, err := parseDeploymentEnvironmentExample(content)
	if err != nil {
		t.Fatalf("parse deployment environment example: %v", err)
	}
	return values
}

func parseDeploymentEnvironmentExample(content string) (map[string]string, error) {
	allowed := make(map[string]struct{}, len(deploymentEnvironmentKeys()))
	for _, key := range deploymentEnvironmentKeys() {
		allowed[key] = struct{}{}
	}

	values := make(map[string]string)
	for lineNumber, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("environment example line %d is not KEY=VALUE", lineNumber+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("environment example line %d has an empty key", lineNumber+1)
		}
		if _, exists := allowed[key]; !exists {
			return nil, fmt.Errorf("environment example line %d uses unapproved key %q", lineNumber+1, key)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("environment example repeats key %q", key)
		}
		if strings.TrimSpace(value) != "" {
			return nil, fmt.Errorf("environment example key %q must have an empty value", key)
		}
		values[key] = ""
	}

	for _, key := range deploymentEnvironmentKeys() {
		if _, exists := values[key]; !exists {
			return nil, fmt.Errorf("environment example is missing approved key %q", key)
		}
	}
	return values, nil
}

func deploymentEnvironmentFixture() string {
	assignments := make([]string, 0, len(deploymentEnvironmentKeys())+1)
	for _, key := range deploymentEnvironmentKeys() {
		assignments = append(assignments, key+"=")
	}
	assignments = append(assignments, "")
	return strings.Join(assignments, "\n")
}

func deploymentEnvironmentKeys() []string {
	return []string{
		EnvProcessingRegionRef,
		EnvProcessingControllerRef,
		EnvProcessingLegalBasisRef,
		EnvSMTPProcessorRef,
		EnvPublicOrigin,
		EnvTrustedProxyCIDRs,
		EnvSMTPTLSMode,
		EnvSMTPHost,
		EnvSMTPServerName,
		EnvSMTPCAPath,
	}
}

func readDeploymentArtifact(t *testing.T, pathElements ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{"..", ".."}, pathElements...)...)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read deployment artifact %s: %v", path, err)
	}
	return string(content)
}

func requireEmptyDeploymentValue(t *testing.T, values map[string]string, key string) {
	t.Helper()

	value, exists := values[key]
	if !exists {
		t.Fatalf("environment example is missing %q", key)
	}
	if value != "" {
		t.Fatalf("environment example key %q has a nonempty value", key)
	}
}

func TestLoadDeploymentConfig_MissingOrBlankRequiredValue_FailsClosedWithoutValueLeak(t *testing.T) {
	valid := validDeploymentValues()
	requiredKeys := []string{
		EnvProcessingRegionRef,
		EnvProcessingControllerRef,
		EnvProcessingLegalBasisRef,
		EnvSMTPProcessorRef,
		EnvPublicOrigin,
		EnvTrustedProxyCIDRs,
		EnvSMTPTLSMode,
		EnvSMTPHost,
		EnvSMTPServerName,
	}

	for _, key := range requiredKeys {
		for _, test := range []struct {
			name  string
			value string
			set   bool
		}{
			{name: "missing"},
			{name: "blank", value: " \t ", set: true},
		} {
			t.Run(key+"/"+test.name, func(t *testing.T) {
				values := mapsClone(valid)
				if test.set {
					values[key] = test.value
				} else {
					delete(values, key)
				}

				_, err := LoadDeploymentConfig(mapLookup(values))
				if err == nil {
					t.Fatal("LoadDeploymentConfig() error = nil, want fail-closed error")
				}
				if !strings.Contains(err.Error(), key) {
					t.Fatalf("LoadDeploymentConfig() error = %q, want key %q", err, key)
				}
				requireErrorDoesNotLeakValues(t, err, valid)
			})
		}
	}
}

func TestLoadDeploymentConfig_ValidValues_ReturnsNormalizedConfig(t *testing.T) {
	values := validDeploymentValues()
	values[EnvPublicOrigin] = " https://Community.Example.Invalid/ "
	values[EnvTrustedProxyCIDRs] = " 192.0.2.9/24, 2001:db8::1/64 "
	values[EnvSMTPHost] = " smtp.example.invalid "
	values[EnvSMTPServerName] = " mail.example.invalid "
	values[EnvSMTPCAPath] = " synthetic-ca-bundle.pem "

	config, err := LoadDeploymentConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadDeploymentConfig() error = %v", err)
	}

	wantPrefixes := []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("2001:db8::/64"),
	}
	if config.ProcessingGate != (ProcessingGateConfig{
		RegionRef:        "synthetic-region-ref",
		ControllerRef:    "synthetic-controller-ref",
		LegalBasisRef:    "synthetic-basis-ref",
		SMTPProcessorRef: "synthetic-smtp-processor-ref",
	}) {
		t.Fatalf("LoadDeploymentConfig() ProcessingGate = %#v, want trimmed references", config.ProcessingGate)
	}
	if config.PublicOrigin != "https://community.example.invalid" {
		t.Fatalf("LoadDeploymentConfig() PublicOrigin = %q, want normalized origin", config.PublicOrigin)
	}
	if !reflect.DeepEqual(config.TrustedProxyCIDRs, wantPrefixes) {
		t.Fatalf("LoadDeploymentConfig() TrustedProxyCIDRs = %v, want %v", config.TrustedProxyCIDRs, wantPrefixes)
	}
	if config.SMTPTLSMode != "implicit_tls" ||
		config.SMTPHost != "smtp.example.invalid" ||
		config.SMTPServerName != "mail.example.invalid" ||
		config.SMTPCAPath != "synthetic-ca-bundle.pem" {
		t.Fatalf("LoadDeploymentConfig() SMTP fields were not trimmed and preserved")
	}
}

func TestLoadDeploymentConfig_HTTPOrigin_NormalizesTrailingSlash(t *testing.T) {
	values := validDeploymentValues()
	values[EnvPublicOrigin] = " http://Community.Example.Invalid/ "

	config, err := LoadDeploymentConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadDeploymentConfig() error = %v", err)
	}
	if config.PublicOrigin != "http://community.example.invalid" {
		t.Fatalf("LoadDeploymentConfig() PublicOrigin = %q, want normalized HTTP origin", config.PublicOrigin)
	}
}

func TestLoadDeploymentConfig_OriginPort_NormalizesCanonicalAuthority(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "HTTPS hostname default port removed",
			value: "https://Community.Example.Invalid:443/",
			want:  "https://community.example.invalid",
		},
		{
			name:  "HTTP hostname default port removed",
			value: "http://Community.Example.Invalid:80/",
			want:  "http://community.example.invalid",
		},
		{
			name:  "hostname nondefault port preserved",
			value: "https://Community.Example.Invalid:8443/",
			want:  "https://community.example.invalid:8443",
		},
		{
			name:  "IPv4 default port removed",
			value: "http://192.0.2.10:80/",
			want:  "http://192.0.2.10",
		},
		{
			name:  "IPv4 nondefault port preserved",
			value: "https://192.0.2.10:8443/",
			want:  "https://192.0.2.10:8443",
		},
		{
			name:  "IPv6 default port removed with brackets preserved",
			value: "https://[2001:DB8::1]:443/",
			want:  "https://[2001:db8::1]",
		},
		{
			name:  "IPv6 nondefault port preserved",
			value: "http://[2001:DB8::1]:8080/",
			want:  "http://[2001:db8::1]:8080",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validDeploymentValues()
			values[EnvPublicOrigin] = test.value

			config, err := LoadDeploymentConfig(mapLookup(values))
			if err != nil {
				t.Fatalf("LoadDeploymentConfig() error = %v", err)
			}
			if config.PublicOrigin != test.want {
				t.Fatalf("LoadDeploymentConfig() PublicOrigin = %q, want %q", config.PublicOrigin, test.want)
			}
		})
	}
}

func TestLoadDeploymentConfig_InvalidOrigin_FailsClosedWithoutValueLeak(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "non HTTP scheme", value: "ftp://origin.example.invalid"},
		{name: "userinfo", value: "https://user@origin.example.invalid"},
		{name: "query", value: "https://origin.example.invalid?mode=test"},
		{name: "fragment", value: "https://origin.example.invalid#fragment"},
		{name: "path", value: "https://origin.example.invalid/path"},
		{name: "missing host", value: "https:///"},
		{name: "empty port", value: "https://origin.example.invalid:/"},
		{name: "nonnumeric port", value: "https://origin.example.invalid:private/"},
		{name: "zero port", value: "https://origin.example.invalid:0/"},
		{name: "out of range port", value: "https://origin.example.invalid:65536/"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validDeploymentValues()
			values[EnvPublicOrigin] = test.value

			_, err := LoadDeploymentConfig(mapLookup(values))
			if err == nil {
				t.Fatal("LoadDeploymentConfig() error = nil, want invalid origin rejected")
			}
			if !strings.Contains(err.Error(), "HAN_PUBLIC_ORIGIN") {
				t.Fatalf("LoadDeploymentConfig() error = %q, want origin key", err)
			}
			if strings.Contains(err.Error(), test.value) {
				t.Fatal("LoadDeploymentConfig() error leaked configured origin")
			}
		})
	}
}

func TestLoadDeploymentConfig_InvalidTrustedProxyCIDRs_FailsClosedWithoutValueLeak(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty item", value: "192.0.2.0/24,,2001:db8::/64"},
		{name: "invalid CIDR", value: "private-proxy-network"},
		{name: "address without prefix", value: "192.0.2.10"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validDeploymentValues()
			values[EnvTrustedProxyCIDRs] = test.value

			_, err := LoadDeploymentConfig(mapLookup(values))
			if err == nil {
				t.Fatal("LoadDeploymentConfig() error = nil, want invalid proxy CIDRs rejected")
			}
			if !strings.Contains(err.Error(), "HAN_TRUSTED_PROXY_CIDRS") {
				t.Fatalf("LoadDeploymentConfig() error = %q, want trusted proxy key", err)
			}
			if strings.Contains(err.Error(), test.value) {
				t.Fatal("LoadDeploymentConfig() error leaked configured proxy CIDRs")
			}
		})
	}
}

func TestLoadDeploymentConfig_InvalidSMTPTLSMode_FailsClosedWithoutValueLeak(t *testing.T) {
	const invalidMode = "opportunistic-private-mode"
	values := validDeploymentValues()
	values[EnvSMTPTLSMode] = invalidMode

	_, err := LoadDeploymentConfig(mapLookup(values))
	if err == nil {
		t.Fatal("LoadDeploymentConfig() error = nil, want invalid SMTP TLS mode rejected")
	}
	if !strings.Contains(err.Error(), "HAN_SMTP_TLS_MODE") {
		t.Fatalf("LoadDeploymentConfig() error = %q, want SMTP TLS key", err)
	}
	if strings.Contains(err.Error(), invalidMode) {
		t.Fatal("LoadDeploymentConfig() error leaked configured SMTP TLS mode")
	}
}

func TestLoadDeploymentConfig_SMTPCAPathMissingOrBlank_UsesSystemRoots(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
	}{
		{name: "missing"},
		{name: "blank", value: " \t ", set: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validDeploymentValues()
			if test.set {
				values[EnvSMTPCAPath] = test.value
			} else {
				delete(values, EnvSMTPCAPath)
			}

			config, err := LoadDeploymentConfig(mapLookup(values))
			if err != nil {
				t.Fatalf("LoadDeploymentConfig() error = %v", err)
			}
			if config.SMTPCAPath != "" {
				t.Fatalf("LoadDeploymentConfig() SMTPCAPath = %q, want system roots marker", config.SMTPCAPath)
			}
		})
	}
}

func validDeploymentValues() map[string]string {
	return map[string]string{
		EnvProcessingRegionRef:     "synthetic-region-ref",
		EnvProcessingControllerRef: "synthetic-controller-ref",
		EnvProcessingLegalBasisRef: "synthetic-basis-ref",
		EnvSMTPProcessorRef:        "synthetic-smtp-processor-ref",
		EnvPublicOrigin:            "https://community.example.invalid",
		EnvTrustedProxyCIDRs:       "192.0.2.0/24",
		EnvSMTPTLSMode:             "implicit_tls",
		EnvSMTPHost:                "smtp.example.invalid",
		EnvSMTPServerName:          "smtp.example.invalid",
		EnvSMTPCAPath:              "",
	}
}

func requireErrorDoesNotLeakValues(t *testing.T, err error, values map[string]string) {
	t.Helper()

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("configuration error leaked a configured value")
		}
	}
}
