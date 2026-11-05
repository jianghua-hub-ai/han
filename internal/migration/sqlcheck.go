package migration

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// rejectTransactionControl prevents a migration from taking ownership of the
// transaction that the runner must commit or roll back.
func rejectTransactionControl(sql string) error {
	var statementTokens []string
	parenthesisDepth := 0
	routineDefinition := false
	pendingAtomicBegin := false
	var atomicBlocks []sqlAtomicBlock

	resetStatement := func() {
		statementTokens = statementTokens[:0]
		routineDefinition = false
		pendingAtomicBegin = false
	}

	for i := 0; i < len(sql); {
		switch {
		case isSQLSpace(sql[i]):
			i++
		case hasSQLPrefix(sql, i, "--"):
			i = skipLineComment(sql, i+2)
		case hasSQLPrefix(sql, i, "/*"):
			next, err := skipBlockComment(sql, i+2)
			if err != nil {
				return err
			}
			i = next
		case isEscapeStringStart(sql, i):
			next, err := skipQuoted(sql, i+2, '\'', true)
			if err != nil {
				return fmt.Errorf("scan escape string: %w", err)
			}
			i = next
		case sql[i] == '\'':
			next, err := skipQuoted(sql, i+1, '\'', false)
			if err != nil {
				return fmt.Errorf("scan SQL string: %w", err)
			}
			i = next
		case sql[i] == '"':
			next, err := skipQuoted(sql, i+1, '"', false)
			if err != nil {
				return fmt.Errorf("scan quoted identifier: %w", err)
			}
			i = next
		case sql[i] == '$':
			if !canStartDollarQuote(sql, i) {
				i++
				continue
			}
			delimiter, contentStart := dollarQuoteDelimiter(sql, i)
			if delimiter == "" {
				i++
				continue
			}
			endOffset := strings.Index(sql[contentStart:], delimiter)
			if endOffset < 0 {
				return fmt.Errorf("scan dollar-quoted body: unterminated delimiter %q", delimiter)
			}
			i = contentStart + endOffset + len(delimiter)
		case sql[i] == '(':
			parenthesisDepth++
			i++
		case sql[i] == ')':
			if parenthesisDepth > 0 {
				parenthesisDepth--
			}
			i++
		case sql[i] == ';':
			if len(atomicBlocks) > 0 {
				pendingAtomicBegin = false
			} else if parenthesisDepth == 0 {
				resetStatement()
			}
			i++
		case isSQLIdentifierStart(sql, i):
			end := sqlIdentifierEnd(sql, i)
			word := asciiSQLKeyword(sql[i:end])
			if len(atomicBlocks) > 0 {
				switch {
				case pendingAtomicBegin && word == "ATOMIC":
					atomicBlocks = append(atomicBlocks, sqlAtomicBlockAtomic)
					pendingAtomicBegin = false
				default:
					pendingAtomicBegin = false
					switch word {
					case "BEGIN":
						pendingAtomicBegin = true
					case "CASE":
						atomicBlocks = append(atomicBlocks, sqlAtomicBlockCase)
					case "END":
						atomicBlocks = atomicBlocks[:len(atomicBlocks)-1]
					}
				}
				i = end
				continue
			}

			if pendingAtomicBegin {
				if word == "ATOMIC" {
					atomicBlocks = append(atomicBlocks, sqlAtomicBlockAtomic)
					pendingAtomicBegin = false
					i = end
					continue
				}
				pendingAtomicBegin = false
			}

			if parenthesisDepth == 0 {
				if len(statementTokens) < 5 {
					statementTokens = append(statementTokens, word)
				}
				if isCreateRoutineStatement(statementTokens) {
					routineDefinition = true
				}
				if routineDefinition && word == "BEGIN" {
					pendingAtomicBegin = true
				}
				if command, found := transactionControlCommand(statementTokens); found {
					return fmt.Errorf("top-level transaction control statement %q is not allowed", command)
				}
			}
			i = end
		default:
			i++
		}
	}

	return nil
}

func isCreateRoutineStatement(tokens []string) bool {
	if len(tokens) == 2 && tokens[0] == "CREATE" {
		return tokens[1] == "FUNCTION" || tokens[1] == "PROCEDURE"
	}
	return len(tokens) == 4 &&
		tokens[0] == "CREATE" &&
		tokens[1] == "OR" &&
		tokens[2] == "REPLACE" &&
		(tokens[3] == "FUNCTION" || tokens[3] == "PROCEDURE")
}

type sqlAtomicBlock uint8

const (
	sqlAtomicBlockAtomic sqlAtomicBlock = iota
	sqlAtomicBlockCase
)

func transactionControlCommand(tokens []string) (string, bool) {
	switch tokens[0] {
	case "BEGIN", "COMMIT", "END", "ROLLBACK", "ABORT", "SAVEPOINT", "RELEASE":
		return tokens[0], true
	case "START":
		if len(tokens) >= 2 && tokens[1] == "TRANSACTION" {
			return "START TRANSACTION", true
		}
	case "PREPARE":
		if len(tokens) >= 2 && tokens[1] == "TRANSACTION" {
			return "PREPARE TRANSACTION", true
		}
	case "SET":
		if len(tokens) >= 2 && tokens[1] == "TRANSACTION" {
			return "SET TRANSACTION", true
		}
		if len(tokens) >= 5 &&
			tokens[1] == "SESSION" &&
			tokens[2] == "CHARACTERISTICS" &&
			tokens[3] == "AS" &&
			tokens[4] == "TRANSACTION" {
			return "SET SESSION CHARACTERISTICS AS TRANSACTION", true
		}
	}
	return "", false
}

func skipLineComment(sql string, i int) int {
	for i < len(sql) && sql[i] != '\n' && sql[i] != '\r' {
		i++
	}
	return i
}

func skipBlockComment(sql string, i int) (int, error) {
	depth := 1
	for i < len(sql) {
		switch {
		case hasSQLPrefix(sql, i, "/*"):
			depth++
			i += 2
		case hasSQLPrefix(sql, i, "*/"):
			depth--
			i += 2
			if depth == 0 {
				return i, nil
			}
		default:
			i++
		}
	}
	return 0, fmt.Errorf("scan block comment: unterminated comment")
}

func skipQuoted(sql string, i int, quote byte, backslashEscapes bool) (int, error) {
	for i < len(sql) {
		if backslashEscapes && sql[i] == '\\' && i+1 < len(sql) {
			i += 2
			continue
		}
		if sql[i] != quote {
			i++
			continue
		}
		if i+1 < len(sql) && sql[i+1] == quote {
			i += 2
			continue
		}
		return i + 1, nil
	}
	return 0, fmt.Errorf("unterminated quoted text")
}

func dollarQuoteDelimiter(sql string, i int) (string, int) {
	if i+1 >= len(sql) {
		return "", 0
	}
	if sql[i+1] == '$' {
		return "$$", i + 2
	}

	first := true
	for end := i + 1; end < len(sql); {
		r, width := utf8.DecodeRuneInString(sql[end:])
		if r == '$' {
			return sql[i : end+width], end + width
		}
		if !isDollarTagRune(r, width, first) {
			return "", 0
		}
		first = false
		end += width
	}
	return "", 0
}

func canStartDollarQuote(sql string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(sql[:i])
	return !isSQLIdentifierRune(r, false)
}

func hasSQLPrefix(sql string, i int, prefix string) bool {
	return len(sql)-i >= len(prefix) && sql[i:i+len(prefix)] == prefix
}

func isEscapeStringStart(sql string, i int) bool {
	if i+1 >= len(sql) || sql[i+1] != '\'' || sql[i] != 'E' && sql[i] != 'e' {
		return false
	}
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(sql[:i])
	return !isSQLIdentifierRune(r, false)
}

func isSQLSpace(char byte) bool {
	switch char {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func isSQLIdentifierStart(sql string, i int) bool {
	r, _ := utf8.DecodeRuneInString(sql[i:])
	return isSQLIdentifierRune(r, true)
}

func sqlIdentifierEnd(sql string, i int) int {
	first := true
	for i < len(sql) {
		r, width := utf8.DecodeRuneInString(sql[i:])
		if !isSQLIdentifierRune(r, first) {
			break
		}
		first = false
		i += width
	}
	return i
}

func isSQLIdentifierRune(r rune, first bool) bool {
	switch {
	case r == '_':
		return true
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case !first && r >= '0' && r <= '9':
		return true
	case !first && r == '$':
		return true
	case r >= 0x80:
		return true
	default:
		return false
	}
}

func asciiSQLKeyword(identifier string) string {
	keyword := make([]byte, len(identifier))
	for i := range len(identifier) {
		char := identifier[i]
		if char >= utf8.RuneSelf {
			return ""
		}
		if char >= 'a' && char <= 'z' {
			char -= 'a' - 'A'
		}
		keyword[i] = char
	}
	return string(keyword)
}

func isDollarTagRune(r rune, width int, first bool) bool {
	return isSQLIdentifierRune(r, first) &&
		r != '$' &&
		(r != utf8.RuneError || width > 1)
}
