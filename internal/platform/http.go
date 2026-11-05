package platform

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TraceMiddleware creates one trusted trace identifier per incoming request.
func TraceMiddleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		traceID, err := NewTraceID()
		if err != nil {
			_ = WriteJSON(response, http.StatusInternalServerError, Envelope{
				Code:    CodePlatformInternalError,
				Message: "服务内部错误",
			})
			return
		}
		ctx, err := ContextWithTraceID(request.Context(), traceID)
		if err != nil {
			_ = WriteJSON(response, http.StatusInternalServerError, Envelope{
				Code:    CodePlatformInternalError,
				Message: "服务内部错误",
			})
			return
		}
		response.Header().Set(TraceHeader, traceID)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

// DecodeJSON strictly decodes one bounded JSON object.
func DecodeJSON(request *http.Request, maxBodyBytes int64, destination any) error {
	if request == nil || request.Body == nil {
		return invalidJSONError(errors.New("request body is required"))
	}
	if maxBodyBytes <= 0 {
		return invalidJSONError(errors.New("JSON body size limit must be positive"))
	}
	if destination == nil {
		return invalidJSONError(errors.New("JSON destination is required"))
	}

	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return invalidJSONError(errors.New("Content-Type must be application/json"))
	}

	body, err := io.ReadAll(io.LimitReader(request.Body, maxBodyBytes+1))
	if err != nil {
		return invalidJSONError(fmt.Errorf("read JSON body: %w", err))
	}
	if int64(len(body)) > maxBodyBytes {
		return invalidJSONError(errors.New("JSON body exceeds size limit"))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return invalidJSONError(errors.New("JSON body is empty"))
	}
	if err := validateSingleJSONObject(body); err != nil {
		return invalidJSONError(err)
	}
	if err := validateExactJSONFields(body, reflect.TypeOf(destination)); err != nil {
		return invalidJSONError(err)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return invalidJSONError(fmt.Errorf("decode JSON body: %w", err))
	}
	if err := requireJSONEOF(decoder); err != nil {
		return invalidJSONError(err)
	}
	return nil
}

// WriteJSON writes one JSON response.
func WriteJSON(response http.ResponseWriter, status int, payload any) error {
	if response == nil {
		return errors.New("HTTP response writer is required")
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	if _, err := response.Write(body.Bytes()); err != nil {
		return fmt.Errorf("write JSON response: %w", err)
	}
	return nil
}

func invalidJSONError(cause error) error {
	return NewPublicError(CodeRequestInvalidJSON, cause)
}

func validateSingleJSONObject(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("validate JSON body: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != json.Delim('{') {
		return errors.New("top-level JSON value must be an object")
	}
	if err := consumeJSONToken(decoder, token); err != nil {
		return fmt.Errorf("validate JSON body: %w", err)
	}
	return requireJSONEOF(decoder)
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	return consumeJSONToken(decoder, token)
}

func consumeJSONToken(decoder *json.Decoder, token json.Token) error {
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}

	switch delim {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := token.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, duplicate := keys[key]; duplicate {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			keys[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if token != json.Delim('}') {
			return errors.New("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if token != json.Delim(']') {
			return errors.New("JSON array is not closed")
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	return nil
}

func validateExactJSONFields(raw []byte, destinationType reflect.Type) error {
	destinationType = indirectType(destinationType)
	if destinationType == nil {
		return errors.New("JSON destination type is required")
	}
	return validateJSONValueAgainstType(raw, destinationType)
}

func validateJSONValueAgainstType(raw []byte, destinationType reflect.Type) error {
	destinationType = indirectType(destinationType)
	if destinationType == nil {
		return nil
	}

	trimmed := bytes.TrimSpace(raw)
	if hasCustomJSONDecoder(destinationType) {
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			return fmt.Errorf("composite JSON is not allowed for custom decoder type %s", destinationType)
		}
		return nil
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	switch destinationType.Kind() {
	case reflect.Struct:
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return nil
		}
		fields, err := exactJSONFields(destinationType)
		if err != nil {
			return err
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &object); err != nil {
			return err
		}
		for name, value := range object {
			fieldType, exists := fields[name]
			if !exists {
				return fmt.Errorf("unknown JSON field %q", name)
			}
			if err := validateJSONValueAgainstType(value, fieldType); err != nil {
				return fmt.Errorf("validate JSON field %q: %w", name, err)
			}
		}
	case reflect.Slice, reflect.Array:
		if len(trimmed) == 0 || trimmed[0] != '[' {
			return nil
		}
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		for index, item := range items {
			if err := validateJSONValueAgainstType(item, destinationType.Elem()); err != nil {
				return fmt.Errorf("validate JSON array item %d: %w", index, err)
			}
		}
	case reflect.Map:
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return nil
		}
		var entries map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return err
		}
		for name, value := range entries {
			if err := validateJSONValueAgainstType(value, destinationType.Elem()); err != nil {
				return fmt.Errorf("validate JSON map entry %q: %w", name, err)
			}
		}
	}
	return nil
}

type jsonFieldCandidate struct {
	valueType reflect.Type
	depth     int
	tagged    bool
}

func exactJSONFields(structType reflect.Type) (map[string]reflect.Type, error) {
	candidates := make(map[string][]jsonFieldCandidate)
	collectExactJSONFields(
		structType,
		0,
		candidates,
		make(map[reflect.Type]bool),
	)

	fields := make(map[string]reflect.Type)
	for name, namedCandidates := range candidates {
		minimumDepth := namedCandidates[0].depth
		for _, candidate := range namedCandidates[1:] {
			if candidate.depth < minimumDepth {
				minimumDepth = candidate.depth
			}
		}

		var shallowest []jsonFieldCandidate
		for _, candidate := range namedCandidates {
			if candidate.depth == minimumDepth {
				shallowest = append(shallowest, candidate)
			}
		}
		if len(shallowest) == 1 {
			fields[name] = shallowest[0].valueType
			continue
		}

		var tagged *jsonFieldCandidate
		taggedCount := 0
		for index := range shallowest {
			if shallowest[index].tagged {
				tagged = &shallowest[index]
				taggedCount++
			}
		}
		if taggedCount == 1 {
			fields[name] = tagged.valueType
		}
	}
	return fields, nil
}

func collectExactJSONFields(
	structType reflect.Type,
	depth int,
	candidates map[string][]jsonFieldCandidate,
	visiting map[reflect.Type]bool,
) {
	structType = indirectType(structType)
	if structType == nil || structType.Kind() != reflect.Struct || hasCustomJSONDecoder(structType) {
		return
	}
	if visiting[structType] {
		return
	}
	visiting[structType] = true
	defer delete(visiting, structType)

	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		tagName, tagged, ignored, embed := jsonFieldName(field)
		if ignored {
			continue
		}

		embeddedType := indirectType(field.Type)
		if field.PkgPath != "" &&
			(!field.Anonymous || embeddedType == nil || embeddedType.Kind() != reflect.Struct) {
			continue
		}
		if (field.Anonymous && tagName == "" || embed && tagName == "") &&
			embeddedType != nil && embeddedType.Kind() == reflect.Struct &&
			!hasCustomJSONDecoder(embeddedType) {
			collectExactJSONFields(embeddedType, depth+1, candidates, visiting)
			continue
		}

		name := tagName
		if name == "" {
			name = field.Name
		}
		candidates[name] = append(candidates[name], jsonFieldCandidate{
			valueType: field.Type,
			depth:     depth,
			tagged:    tagged,
		})
	}
}

func jsonFieldName(field reflect.StructField) (string, bool, bool, bool) {
	tag, exists := field.Tag.Lookup("json")
	if !exists {
		return "", false, false, false
	}
	if tag == "-" {
		return "", false, true, false
	}

	name, options := bestEffortJSONTagName(tag)
	embed := false
	for _, option := range strings.Split(options, ",") {
		if option == "embed" {
			embed = true
		}
	}
	return name, name != "", false, embed
}

func bestEffortJSONTagName(tag string) (string, string) {
	if tag == "" || !utf8.ValidString(tag) {
		return "", ""
	}

	reserved := strings.IndexAny(tag, ",\\'\"`")
	if reserved < 0 {
		return tag, ""
	}
	if tag[reserved] == ',' {
		return tag[:reserved], tag[reserved+1:]
	}

	first, _ := utf8.DecodeRuneInString(tag)
	if first != '_' && !unicode.IsLetter(first) {
		return "", ""
	}
	end := 0
	for end < len(tag) {
		current, size := utf8.DecodeRuneInString(tag[end:])
		if current != '_' && !unicode.IsLetter(current) && !unicode.IsNumber(current) {
			break
		}
		end += size
	}
	return tag[:end], ""
}

func indirectType(valueType reflect.Type) reflect.Type {
	for valueType != nil && valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	return valueType
}

func hasCustomJSONDecoder(valueType reflect.Type) bool {
	jsonUnmarshaler := reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textUnmarshaler := reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	return valueType.Implements(jsonUnmarshaler) ||
		reflect.PointerTo(valueType).Implements(jsonUnmarshaler) ||
		valueType.Implements(textUnmarshaler) ||
		reflect.PointerTo(valueType).Implements(textUnmarshaler)
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("JSON body contains multiple values")
	}
	return fmt.Errorf("decode trailing JSON data: %w", err)
}
