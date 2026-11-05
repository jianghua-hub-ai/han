// Package security provides credential policy and hashing primitives.
package security

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

const minimumPasswordCharacters = 8

var (
	// ErrPasswordTooShort indicates that a password has fewer than eight characters.
	ErrPasswordTooShort = errors.New("password is too short")
	// ErrPasswordRequiresLetter indicates that a password contains no letter.
	ErrPasswordRequiresLetter = errors.New("password requires a letter")
	// ErrPasswordRequiresDigit indicates that a password contains no digit.
	ErrPasswordRequiresDigit = errors.New("password requires a digit")
)

// PasswordPolicy validates the confirmed registration password rules.
type PasswordPolicy struct{}

// Validate checks a password without including it in returned errors.
func (PasswordPolicy) Validate(password string) error {
	if utf8.RuneCountInString(password) < minimumPasswordCharacters {
		return ErrPasswordTooShort
	}

	var hasLetter, hasDigit bool
	for _, character := range password {
		hasLetter = hasLetter || unicode.IsLetter(character)
		hasDigit = hasDigit || unicode.IsDigit(character)
	}
	if !hasLetter {
		return ErrPasswordRequiresLetter
	}
	if !hasDigit {
		return ErrPasswordRequiresDigit
	}
	return nil
}
