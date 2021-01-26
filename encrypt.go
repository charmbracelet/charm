package charm

// EncryptKey is the symmetric key used to encrypt data for a Charm user. An
// encrypt key will be encoded for every public key associated with a user's
// Charm account.
type EncryptKey struct {
	GlobalID  string `json:"global_id"`
	Key       string `json:"key"`
	PublicKey string `json:"public_key,omitempty"`
}
