package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

var (
	encryptCmd = &cobra.Command{
		Use:    "encrypt",
		Hidden: false,
		Short:  "Encrypt stdin with your Charm account encryption key",
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			b, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			enc, gid, err := cc.Encrypt(b)
			if err != nil {
				return err
			}
			cf := CryptFile{
				EncryptKey: gid,
				Data:       base64.StdEncoding.EncodeToString(enc),
			}
			out, err := json.Marshal(cf)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}
)
