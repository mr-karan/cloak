package main

import "fmt"

const (
	MaxAccessCount   = 30
	MaxExpiry        = 86400 * 7 // 7 days.
	MinPassphraseLen = 6
	MaxPassphraseLen = 64
)

func (enc *EncryptPayload) Validate() error {
	// Max expiry shouldn't exceed 7 days
	if enc.Expiry > MaxExpiry {
		return fmt.Errorf("Expiry exceeds the max allowed limit: %d", MaxExpiry)
	}
	if enc.AccessCount > MaxExpiry {
		return fmt.Errorf("Access Count exceeds the max allowed limit: %d", MaxAccessCount)
	}
	if err := isValidPassphrase(enc.Passphrase); err != nil {
		return err
	}
	return nil
}

func (enc *LookupPayload) Validate() error {
	if err := isValidPassphrase(enc.Passphrase); err != nil {
		return err
	}
	return nil
}

func isValidPassphrase(p string) error {
	if (len(p) > MaxPassphraseLen) || (len(p) < MinPassphraseLen) {
		return fmt.Errorf("Passphrase length should be between %d and %d characters", MinPassphraseLen, MaxPassphraseLen)
	}
	return nil
}
