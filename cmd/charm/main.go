package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/meowgorithm/babyenv"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var ErrMissingSSHAuth = errors.New("missing ssh auth")

type Config struct {
	Host       string `env:"CHARM_ID_HOST" default:"id.dev.charm.sh"`
	Port       int    `env:"CHARM_ID_PORT" default:"5555"`
	SSHKeyPath string `env:"CHARM_SSH_KEY_PATH" default:"~/.ssh/id_dsa"`
}

type CharmClient struct {
	Config      *Config
	AgentClient *ssh.Client
	AuthMethod  ssh.AuthMethod
	SSHKeyPath  string
}

func ConnectCharm(cfg Config) (*CharmClient, error) {
	var sshCfg *ssh.ClientConfig
	am, err := agentAuthMethod()
	if err == nil {
		sshCfg = &ssh.ClientConfig{
			User:            "charm",
			Auth:            []ssh.AuthMethod{am},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	} else {
		var pkam ssh.AuthMethod
		pkam, err = publicKeyAuthMethod(cfg.SSHKeyPath)
		if err != nil {
			pkam, err = publicKeyAuthMethod("~/.ssh/id_rsa")
			if err != nil {
				return nil, ErrMissingSSHAuth
			}
		}
		sshCfg = &ssh.ClientConfig{
			User:            "charm",
			Auth:            []ssh.AuthMethod{pkam},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	}
	sshc, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		return nil, err
	}
	return &CharmClient{
		AgentClient: sshc,
		Config:      &cfg,
	}, nil
}

func (cc *CharmClient) Close() {
	cc.AgentClient.Close()
}

func (cc *CharmClient) JWT() (string, error) {
	s, err := cc.AgentClient.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	id, err := s.Output("jwt")
	if err != nil {
		return "", err
	}
	return string(id), nil
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

func publicKeyAuthMethod(kp string) (ssh.AuthMethod, error) {
	keyPath, err := homedir.Expand(kp)
	if err != nil {
		return nil, err
	}
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func agentAuthMethod() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}
	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

func main() {
	var cfg Config
	if err := babyenv.Parse(&cfg); err != nil {
		log.Fatal(err)
	}
	cc, err := ConnectCharm(cfg)
	if err == ErrMissingSSHAuth {
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
