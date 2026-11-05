package main

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errAmbientPostgresConfig       = errors.New("ambient PostgreSQL configuration is not allowed")
	errExternalDatabaseConfig      = errors.New("external PostgreSQL configuration source is not allowed")
	errInvalidDatabaseDSN          = errors.New("invalid database DSN")
	errRoleSwitchingDatabaseOption = errors.New(
		"database DSN options must not change the session role",
	)
)

var postgresEnvironmentKeys = []string{
	"PGHOST",
	"PGPORT",
	"PGDATABASE",
	"PGUSER",
	"PGPASSWORD",
	"PGPASSFILE",
	"PGAPPNAME",
	"PGCONNECT_TIMEOUT",
	"PGSSLMODE",
	"PGSSLKEY",
	"PGSSLCERT",
	"PGSSLSNI",
	"PGSSLROOTCERT",
	"PGSSLPASSWORD",
	"PGSSLNEGOTIATION",
	"PGTARGETSESSIONATTRS",
	"PGSERVICE",
	"PGSERVICEFILE",
	"PGTZ",
	"PGOPTIONS",
	"PGMINPROTOCOLVERSION",
	"PGMAXPROTOCOLVERSION",
	"PGCHANNELBINDING",
	"PGREQUIREAUTH",
}

var externalDatabaseConfigKeys = map[string]struct{}{
	"passfile":    {},
	"service":     {},
	"servicefile": {},
	"sslcert":     {},
	"sslkey":      {},
	"sslrootcert": {},
}

var roleSwitchingDatabaseOption = regexp.MustCompile(`(?i)(^|[\s"'])(?:(?:-c|--)\s*)?(?:role|session_authorization)\s*=`)

func parseDatabasePoolConfig(dsn string, getenv func(string) string) (*pgxpool.Config, error) {
	if strings.TrimSpace(dsn) == "" || getenv == nil {
		return nil, errInvalidDatabaseDSN
	}
	for _, key := range postgresEnvironmentKeys {
		if getenv(key) != "" {
			return nil, errAmbientPostgresConfig
		}
	}

	isolatedDSN, err := isolateDatabaseDSN(dsn)
	if err != nil {
		return nil, err
	}
	config, err := pgxpool.ParseConfig(isolatedDSN)
	if err != nil {
		// pgx parse errors can contain a best-effort redaction of the input.
		// Do not retain that error where a malformed DSN may contain a secret.
		return nil, errInvalidDatabaseDSN
	}
	return config, nil
}

func isolateDatabaseDSN(dsn string) (string, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		queryStart, err := databaseURLQueryStart(dsn)
		if err != nil {
			return "", err
		}
		if hasExternalURLConfigKey(dsn, queryStart) {
			return "", errExternalDatabaseConfig
		}
		if hasRoleSwitchingURLSetting(dsn, queryStart) {
			return "", errRoleSwitchingDatabaseOption
		}
		separator := "?"
		if queryStart >= 0 {
			separator = "&"
		}
		return dsn + separator + "passfile=&servicefile=&sslcert=&sslkey=&sslrootcert=", nil
	}

	settings, err := keywordDatabaseConfigSettings(dsn)
	if err != nil {
		return "", errInvalidDatabaseDSN
	}
	for _, setting := range settings {
		if _, found := externalDatabaseConfigKeys[setting.key]; found {
			return "", errExternalDatabaseConfig
		}
		if strings.EqualFold(setting.key, "role") ||
			strings.EqualFold(setting.key, "session_authorization") {
			return "", errRoleSwitchingDatabaseOption
		}
		if strings.EqualFold(setting.key, "options") &&
			roleSwitchingDatabaseOption.MatchString(setting.value) {
			return "", errRoleSwitchingDatabaseOption
		}
	}
	return dsn + " passfile='' servicefile='' sslcert='' sslkey='' sslrootcert=''", nil
}

func databaseURLQueryStart(dsn string) (int, error) {
	schemeEnd := strings.Index(dsn, "://") + len("://")
	remainder := dsn[schemeEnd:]
	if delimiter := strings.IndexAny(remainder, "@/"); delimiter >= 0 && remainder[delimiter] == '@' {
		if strings.IndexByte(remainder[:delimiter], '?') >= 0 {
			return -1, errInvalidDatabaseDSN
		}
	}
	queryOffset := strings.IndexByte(remainder, '?')
	if queryOffset < 0 {
		return -1, nil
	}
	return schemeEnd + queryOffset, nil
}

func hasExternalURLConfigKey(dsn string, queryStart int) bool {
	if queryStart < 0 {
		return false
	}
	for _, pair := range strings.Split(dsn[queryStart+1:], "&") {
		rawKey, _, found := strings.Cut(pair, "=")
		if !found {
			continue
		}
		key, err := url.PathUnescape(rawKey)
		if err != nil {
			continue
		}
		if _, found := externalDatabaseConfigKeys[strings.Trim(key, " ")]; found {
			return true
		}
	}
	return false
}

func hasRoleSwitchingURLSetting(dsn string, queryStart int) bool {
	if queryStart < 0 {
		return false
	}
	for _, pair := range strings.Split(dsn[queryStart+1:], "&") {
		rawKey, rawValue, hasValue := strings.Cut(pair, "=")
		key, err := url.QueryUnescape(rawKey)
		if err != nil {
			return true
		}
		key = strings.TrimSpace(key)
		if strings.EqualFold(key, "role") ||
			strings.EqualFold(key, "session_authorization") {
			return true
		}
		if !strings.EqualFold(key, "options") || !hasValue {
			continue
		}
		value, err := url.QueryUnescape(rawValue)
		if err != nil {
			return true
		}
		if roleSwitchingDatabaseOption.MatchString(value) {
			return true
		}
	}
	return false
}

type databaseConfigSetting struct {
	key   string
	value string
}

func keywordDatabaseConfigSettings(dsn string) ([]databaseConfigSetting, error) {
	if strings.IndexByte(dsn, 0) >= 0 {
		return nil, errInvalidDatabaseDSN
	}

	var settings []databaseConfigSetting
	remaining := trimDatabaseConfigSpace(dsn)
	for remaining != "" {
		equal := strings.IndexByte(remaining, '=')
		if equal < 0 {
			return nil, errInvalidDatabaseDSN
		}
		key := strings.Trim(remaining[:equal], " \t\n\r\v\f")
		if key == "" || strings.IndexAny(key, " \t\n\r\v\f") >= 0 {
			return nil, errInvalidDatabaseDSN
		}
		remaining = trimDatabaseConfigSpace(remaining[equal+1:])
		if remaining == "" {
			settings = append(settings, databaseConfigSetting{key: key})
			continue
		}
		if remaining[0] == '\'' {
			end, ok := quotedDatabaseConfigValueEnd(remaining[1:])
			if !ok {
				return nil, errInvalidDatabaseDSN
			}
			settings = append(settings, databaseConfigSetting{
				key:   key,
				value: unescapeDatabaseConfigValue(remaining[1 : end+1]),
			})
			remaining = trimDatabaseConfigSpace(remaining[end+2:])
			continue
		}
		end := unquotedDatabaseConfigValueEnd(remaining)
		settings = append(settings, databaseConfigSetting{
			key:   key,
			value: unescapeDatabaseConfigValue(remaining[:end]),
		})
		remaining = trimDatabaseConfigSpace(remaining[end:])
	}
	return settings, nil
}

func keywordDatabaseConfigKeys(dsn string) ([]string, error) {
	settings, err := keywordDatabaseConfigSettings(dsn)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(settings))
	for index, setting := range settings {
		keys[index] = setting.key
	}
	return keys, nil
}

func unescapeDatabaseConfigValue(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	escaped := false
	for index := 0; index < len(value); index++ {
		char := value[index]
		if escaped {
			builder.WriteByte(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			continue
		}
		builder.WriteByte(char)
	}
	if escaped {
		builder.WriteByte('\\')
	}
	return builder.String()
}

func quotedDatabaseConfigValueEnd(value string) (int, bool) {
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\':
			i++
			if i == len(value) {
				return 0, false
			}
		case '\'':
			return i, true
		}
	}
	return 0, false
}

func unquotedDatabaseConfigValueEnd(value string) int {
	for i := 0; i < len(value); i++ {
		if isDatabaseConfigSpace(value[i]) {
			return i
		}
		if value[i] == '\\' {
			i++
			if i == len(value) {
				return i
			}
		}
	}
	return len(value)
}

func trimDatabaseConfigSpace(value string) string {
	return strings.TrimLeft(value, " \t\n\r\v\f")
}

func isDatabaseConfigSpace(char byte) bool {
	switch char {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}
