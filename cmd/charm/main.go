package main

import (
	"flag"
	"log"

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
	jwt, err := cc.JWT()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", jwt)
}
