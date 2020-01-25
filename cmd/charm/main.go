package main

import (
	"fmt"
	"log"
	"net"
	"os"
	gouser "os/user"

	"github.com/meowgorithm/babyenv"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Config struct {
	Host string `env:"CHARM_ID_HOST" default:"id.dev.charm.sh"`
	Port int    `env:"CHARM_ID_PORT" default:"5555"`
}

type CharmClient struct {
	Config      *Config
	AgentClient *ssh.Client
	Agent       agent.Agent
	KeyPath     string
}

type AccountKey struct {
	CharmID   string
	PublicKey ssh.PublicKey
	Agent     bool
}

func NewCharmClient() *CharmClient {
	var cfg Config
	if err := babyenv.Parse(&cfg); err != nil {
		log.Fatalf("could not get environment vars: %v", err)
	}
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	agentClient := agent.NewClient(conn)

	ud, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	kp := fmt.Sprintf("%s/.ssh", ud)
	e, err := fileExists(kp)
	if err != nil {
		log.Fatal(err)
	}
	if !e {
		log.Fatal("missing ssh directory at ~/.ssh")
	}

	u, err := gouser.Current()
	if err != nil {
		log.Fatal(err)
	}

	sshCfg := &ssh.ClientConfig{
		User: u.Name,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agentClient.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshc, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		log.Fatal(err)
	}

	return &CharmClient{
		AgentClient: sshc,
		Agent:       agentClient,
		KeyPath:     kp,
		Config:      &cfg,
	}
}

func (cc *CharmClient) Close() {
	cc.AgentClient.Close()
}

func (cc *CharmClient) AgentKeys() ([]*AccountKey, error) {
	var acs []*AccountKey
	ks, err := cc.Agent.List()
	if err != nil {
		return nil, err
	}
	for _, k := range ks {
		ak := &AccountKey{
			PublicKey: k,
			Agent:     true,
		}
		acs = append(acs, ak)
	}
	return acs, nil

}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func main() {
	cc := NewCharmClient()
	defer cc.Close()
	s, err := cc.AgentClient.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	id, err := s.Output("id")
	s.Close()
	log.Printf("ID: %s", string(id))
}
