package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
	"github.com/spf13/cobra"
)

var (
	decryptCmd = &cobra.Command{
		Use:     "decrypt",
		Hidden:  false,
		Short:   "Decrypt stdin with your Charm account encryption key",
		Long:    formatLong(fmt.Sprintf("%s stdin with your Charm account encryption key.", common.Keyword("Decrypt"))),
		Example: indent.String("charm decrypt < encrypted_data.json\ncat encrypted_data.json | charm decrypt", indentBy),
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var r io.Reader
			var err error

			switch len(args) {
			case 0:
				r = os.Stdin
			default:
				f, err := os.Open(args[0])
				defer f.Close()
				r = f
				if err != nil {
					return err
				}
			}

			cf := &CryptFile{}
			jd := json.NewDecoder(r)
			err = jd.Decode(cf)
			if err != nil {
				return err
			}

			d, err := base64.StdEncoding.DecodeString(cf.Data)
			if err != nil {
				return err
			}
			cc := initCharmClient()
			out, err := cc.Decrypt(cf.EncryptKey, d)
			if err != nil {
				return err
			}
			fmt.Printf("%s", string(out))
			return nil
		},
	}
)
