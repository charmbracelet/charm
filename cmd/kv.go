package cmd

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/charm/kv"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/dgraph-io/badger/v3"
	"github.com/spf13/cobra"
)

var (
	reverseIterate   bool
	keysIterate      bool
	valuesIterate    bool
	showBinary       bool
	delimiterIterate string

	// KVCmd is the cobra.Command for a user to use the Charm key value store.
	KVCmd = &cobra.Command{
		Use:    "kv",
		Hidden: false,
		Short:  "Use the Charm key value store.",
		Long:   paragraph(fmt.Sprintf("Commands to set, get and delete data from your Charm Cloud backed key value store.")),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	kvSetCmd = &cobra.Command{
		Use:    "set KEY[@DB] VALUE",
		Hidden: false,
		Short:  "Set a value for a key with an optional @ db.",
		Args:   cobra.MaximumNArgs(2),
		RunE:   kvSet,
	}

	kvGetCmd = &cobra.Command{
		Use:    "get KEY[@DB]",
		Hidden: false,
		Short:  "Get a value for a key with an optional @ db.",
		Args:   cobra.ExactArgs(1),
		RunE:   kvGet,
	}

	kvDeleteCmd = &cobra.Command{
		Use:    "delete KEY[@DB]",
		Hidden: false,
		Short:  "Delete a key with an optional @ db.",
		Args:   cobra.ExactArgs(1),
		RunE:   kvDelete,
	}

	kvListCmd = &cobra.Command{
		Use:    "list [@DB]",
		Hidden: false,
		Short:  "List all key value pairs with an optional @ db.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   kvList,
	}

	kvSyncCmd = &cobra.Command{
		Use:    "sync [@DB]",
		Hidden: false,
		Short:  "Sync local db with latest Charm Cloud db.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   kvSync,
	}

	kvResetCmd = &cobra.Command{
		Use:    "reset [@DB]",
		Hidden: false,
		Short:  "Delete local db and pull down fresh copy from Charm Cloud.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   kvReset,
	}
)

func kvSet(cmd *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	if len(args) == 2 {
		return db.Set(k, []byte(args[1]))
	}
	return db.SetReader(k, os.Stdin)
}

func kvGet(cmd *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	v, err := db.Get(k)
	if err != nil {
		return err
	}
	printFromKV("%s", v)
	return nil
}

func kvDelete(cmd *cobra.Command, args []string) error {
	k, n, err := keyParser(args[0])
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	return db.Delete(k)
}

func kvList(cmd *cobra.Command, args []string) error {
	var k string
	var pf string
	if keysIterate || valuesIterate {
		pf = "%s\n"
	} else {
		pf = fmt.Sprintf("%%s%s%%s\n", delimiterIterate)
	}
	if len(args) == 1 {
		k = args[0]
	}
	_, n, err := keyParser(k)
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	db.Sync()
	return db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		opts.Reverse = reverseIterate
		if keysIterate {
			opts.PrefetchValues = false
		}
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if keysIterate {
				printFromKV(pf, k)
				continue
			}
			err := item.Value(func(v []byte) error {
				if valuesIterate {
					printFromKV(pf, v)
				} else {
					printFromKV(pf, k, v)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func kvSync(cmd *cobra.Command, args []string) error {
	n, err := nameFromArgs(args)
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	return db.Sync()
}

func kvReset(cmd *cobra.Command, args []string) error {
	n, err := nameFromArgs(args)
	if err != nil {
		return err
	}
	db, err := openKV(n)
	if err != nil {
		return err
	}
	return db.Reset()
}

func nameFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	_, n, err := keyParser(args[0])
	if err != nil {
		return "", err
	}
	return n, nil
}

func printFromKV(pf string, vs ...[]byte) {
	nb := "(omitted binary data)"
	fvs := make([]interface{}, 0)
	for _, v := range vs {
		if common.IsTTY() && !showBinary && !utf8.Valid(v) {
			fvs = append(fvs, nb)
		} else {
			fvs = append(fvs, string(v))
		}
	}
	fmt.Printf(pf, fvs...)
	if common.IsTTY() && !strings.HasSuffix(pf, "\n") {
		fmt.Println()
	}
}

func keyParser(k string) ([]byte, string, error) {
	var key, db string
	ps := strings.Split(k, "@")
	switch len(ps) {
	case 1:
		key = strings.ToLower(ps[0])
	case 2:
		key = strings.ToLower(ps[0])
		db = strings.ToLower(ps[1])
	default:
		return nil, "", fmt.Errorf("bad key format, use KEY@DB")
	}
	return []byte(key), db, nil
}

func openKV(name string) (*kv.KV, error) {
	if name == "" {
		name = "charm.sh.kv.user.default"
	}
	return kv.OpenWithDefaults(name)
}

func init() {
	kvListCmd.Flags().BoolVarP(&reverseIterate, "reverse", "r", false, "list in reverse lexicographic order")
	kvListCmd.Flags().BoolVarP(&keysIterate, "keys-only", "k", false, "only print keys and don't fetch values from the db")
	kvListCmd.Flags().BoolVarP(&valuesIterate, "values-only", "v", false, "only print values")
	kvListCmd.Flags().BoolVarP(&showBinary, "show-binary", "b", false, "print binary values")
	kvGetCmd.Flags().BoolVarP(&showBinary, "show-binary", "b", false, "print binary values")
	kvListCmd.Flags().StringVarP(&delimiterIterate, "delimiter", "d", "\t", "delimiter to separate keys and values")

	KVCmd.AddCommand(kvGetCmd)
	KVCmd.AddCommand(kvSetCmd)
	KVCmd.AddCommand(kvDeleteCmd)
	KVCmd.AddCommand(kvListCmd)
	KVCmd.AddCommand(kvSyncCmd)
	KVCmd.AddCommand(kvResetCmd)
}
