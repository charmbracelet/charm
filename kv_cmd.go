package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/spf13/cobra"
)

var (
	kvCmd = &cobra.Command{
		Use:    "kv",
		Hidden: false,
		Short:  "Use the Charm key value store.",
		Long:   formatLong(fmt.Sprintf("Commands to set, get and delete data from your Charm Cloud backed key value store.")),
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

	kvKeysCmd = &cobra.Command{
		Use:    "keys [@DB]",
		Hidden: false,
		Short:  "List all keys with an optional @ db.",
		Args:   cobra.MaximumNArgs(1),
		RunE:   kvKeys,
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
	fmt.Println(string(v))
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

func kvKeys(cmd *cobra.Command, args []string) error {
	var k string
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
	ks, err := db.Keys()
	if err != nil {
		panic(err)
	}
	for _, k := range ks {
		fmt.Println(string(k))
	}
	return nil
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
	dd, err := client.DataPath()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = "charm.sh.kv.user.default"
	}
	return kv.OpenWithDefaults(name, fmt.Sprintf("%s/kv", dd))
}

func init() {
	kvCmd.AddCommand(kvGetCmd)
	kvCmd.AddCommand(kvSetCmd)
	kvCmd.AddCommand(kvDeleteCmd)
	kvCmd.AddCommand(kvKeysCmd)
	kvCmd.AddCommand(kvSyncCmd)
	kvCmd.AddCommand(kvResetCmd)
}
