package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/charm"
)

func main() {
	i := flag.String("i", "", "identity file (ssh key) path")
	flag.Parse()
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	if *i != "" {
		cfg.SSHKeyPath = *i
	}
	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		log.Fatal("Missing ssh key. Run `ssh-keygen` to make one or set the `CHARM_SSH_KEY_PATH` env var to your private key path.")
	}
	if err != nil {
		log.Fatal(err)
	}
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}
	switch args[0] {
	case "name":
		if len(args) != 2 {
			log.Fatal("Usage: charm name USERNAME")
		}
		n := args[1]
		u, err := cc.SetName(n)
		if err == charm.ErrNameTaken {
			fmt.Printf("User name '%s' is already taken. Try another!\n", n)
			os.Exit(1)
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("@%s ID: %s\n", u.Name, u.CharmID)
	case "jwt":
		jwt, err := cc.JWT()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s", jwt)
	case "id":
		id, err := cc.ID()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s", id)
	case "keys":
		ak, err := cc.AuthorizedKeys()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s", ak)
	case "link":
		switch len(args) {
		case 1:
			lr, err := cc.LinkGen()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(lr)
		case 2:
			lr, err := cc.Link(args[1])
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(lr)
		default:
			log.Fatal("Bad link command")
		}
	default:
		fmt.Printf("'%s' is not a valid command", args[0])
		os.Exit(1)
	}
}
