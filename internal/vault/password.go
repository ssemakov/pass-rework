package vault

import "errors"

var ErrMasterFormat = errors.New("that master passphrase uses a reserved format; choose another passphrase")

// SQLCipher interprets exact x'hex' key specifications as raw keys, bypassing
// password PBKDF2 even though cipher_settings still reports kdf_iter=256000.
// Normal vault passwords must take the passphrase path. Legacy read-only
// import/preservation deliberately retains upstream raw-key compatibility.
func validateMasterPassword(key []byte) error {
	if len(key) == 0 || len(key) > 1024*1024 {
		return ErrInput
	}
	if len(key) != 67 && len(key) != 99 && len(key) != 163 {
		return nil
	}
	if (key[0] != 'x' && key[0] != 'X') || key[1] != '\'' || key[len(key)-1] != '\'' {
		return nil
	}
	for _, b := range key[2 : len(key)-1] {
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')) {
			return nil
		}
	}
	return ErrMasterFormat
}
