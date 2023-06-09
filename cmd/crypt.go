package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/charm/crypt"
	"github.com/spf13/cobra"
)

var (
	// CryptCmd is the cobra.Command to manage encryption and decryption for a user.
	CryptCmd = &cobra.Command{
		Use:    "crypt",
		Hidden: false,
		Short:  "Use Charm encryption.",
		Long:   styles.Paragraph.Render("Commands to encrypt and decrypt data with your Charm Cloud encryption keys."),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cryptEncryptCmd = &cobra.Command{
		Use:    "encrypt",
		Hidden: false,
		Short:  "Encrypt stdin with your Charm account encryption key",
		Args:   cobra.NoArgs,
		RunE:   cryptEncrypt,
	}

	cryptDecryptCmd = &cobra.Command{
		Use:    "decrypt",
		Hidden: false,
		Short:  "Decrypt stdin with your Charm account encryption key",
		Args:   cobra.RangeArgs(0, 1),
		RunE:   cryptDecrypt,
	}

	cryptEncryptLookupCmd = &cobra.Command{
		Use:    "encrypt-lookup",
		Hidden: false,
		Short:  "Encrypt arg deterministically",
		Args:   cobra.ExactArgs(1),
		RunE:   cryptEncryptLookup,
	}

	cryptDecryptLookupCmd = &cobra.Command{
		Use:    "decrypt-lookup",
		Hidden: false,
		Short:  "Decrypt arg deterministically",
		Args:   cobra.ExactArgs(1),
		RunE:   cryptDecryptLookup,
	}
)

type cryptFile struct {
	Data string `json:"data"`
}

func cryptEncrypt(_ *cobra.Command, _ []string) error {
	cr, err := crypt.NewCrypt()
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	eb, err := cr.NewEncryptedWriter(buf)
	if err != nil {
		return err
	}
	_, err = io.Copy(eb, os.Stdin)
	if err != nil {
		return err
	}
	eb.Close() // nolint:errcheck
	cf := cryptFile{
		Data: base64.StdEncoding.EncodeToString(buf.Bytes()),
	}
	out, err := json.Marshal(cf)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func cryptDecrypt(_ *cobra.Command, args []string) error {
	var r io.Reader
	cr, err := crypt.NewCrypt()
	if err != nil {
		return err
	}
	switch len(args) {
	case 0:
		r = os.Stdin
	default:
		f, err := os.Open(args[0])
		defer f.Close() // nolint:errcheck
		r = f
		if err != nil {
			return err
		}
	}
	cf := &cryptFile{}
	jd := json.NewDecoder(r)
	err = jd.Decode(cf)
	if err != nil {
		return err
	}
	d, err := base64.StdEncoding.DecodeString(cf.Data)
	if err != nil {
		return err
	}
	br := bytes.NewReader(d)
	deb, err := cr.NewDecryptedReader(br)
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, deb)
	if err != nil {
		return err
	}
	return nil
}

func cryptEncryptLookup(_ *cobra.Command, args []string) error {
	cr, err := crypt.NewCrypt()
	if err != nil {
		return err
	}
	ct, err := cr.EncryptLookupField(args[0])
	if err != nil {
		return err
	}
	fmt.Println(ct)
	return nil
}

func cryptDecryptLookup(_ *cobra.Command, args []string) error {
	cr, err := crypt.NewCrypt()
	if err != nil {
		return err
	}
	pt, err := cr.DecryptLookupField(args[0])
	if err != nil {
		return err
	}
	fmt.Println(pt)
	return nil
}

func init() {
	CryptCmd.AddCommand(cryptEncryptCmd)
	CryptCmd.AddCommand(cryptDecryptCmd)
	CryptCmd.AddCommand(cryptEncryptLookupCmd)
	CryptCmd.AddCommand(cryptDecryptLookupCmd)
}
