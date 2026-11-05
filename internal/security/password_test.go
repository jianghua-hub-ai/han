package security

import (
	"errors"
	"strings"
	"testing"
)

func TestPasswordPolicy_Validate_EnforcesConfirmedRules(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "shorter than eight characters", password: "abc1234", wantErr: ErrPasswordTooShort},
		{name: "missing letter", password: "12345678", wantErr: ErrPasswordRequiresLetter},
		{name: "missing digit", password: "abcdefgh", wantErr: ErrPasswordRequiresDigit},
		{name: "exactly eight valid characters", password: "abcd1234"},
		{name: "unicode letters count as characters", password: "密码abcd12"},
	}

	var policy PasswordPolicy
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := policy.Validate(test.password)
			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, test.wantErr)
			}
			if strings.Contains(err.Error(), test.password) {
				t.Fatal("Validate() error contains the password")
			}
		})
	}
}
