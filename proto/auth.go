package proto

// Auth is the response to an authenticated connection. It contains tokens and
// keys required to access Charm Cloud services.
type Auth struct {
	JWT         string        `json:"jwt"`
	ID          string        `json:"charm_id"`
	HTTPScheme  string        `json:"http_scheme"`
	PublicKey   string        `json:"public_key,omitempty"`
	EncryptKeys []*EncryptKey `json:"encrypt_keys,omitempty"`
}
