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
	cc, err := charm.ConnectCharm(cfg)
	if err == charm.ErrMissingSSHAuth {
		log.Fatal("Missing ssh key. Run `ssh-keygen` to make one or set the `CHARM_SSH_KEY_PATH` env var to your private key path.")
	}
	if err != nil {
		log.Fatal(err)
	}
	defer cc.Close()
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
	} else {
		switch args[0] {
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
			log.Fatalf("not implemented yet")
		case "link":
			log.Fatalf("not implemented yet")
		default:
			fmt.Printf("'%s' is not a valid command", args[0])
			os.Exit(1)
		}
	}
}
