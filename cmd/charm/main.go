package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/link"
	"github.com/charmbracelet/tea"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/spf13/cobra"
)

const (
	wrapAt   = 78
	indentBy = 2
)

type TermLinkHandler struct{}

func (th *TermLinkHandler) TokenCreated(l *charm.Link) {
	fmt.Printf("To link a machine, run: \n\n> charm link %s\n", l.Token)
}

func (th *TermLinkHandler) TokenSent(l *charm.Link) {
	fmt.Println("Linking...")
}

func (th *TermLinkHandler) ValidToken(l *charm.Link) {
	fmt.Println("Valid token")
}

func (th *TermLinkHandler) InvalidToken(l *charm.Link) {
	fmt.Println("That token looks invalid.")
}

func (th *TermLinkHandler) Request(l *charm.Link) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Does this look right? (yes/no)\n\n%s\nIP: %s\n", l.RequestPubKey, l.RequestAddr)
	conf, _ := reader.ReadString('\n')
	if strings.ToLower(conf) == "yes\n" {
		return true
	}
	return false
}

func (th *TermLinkHandler) RequestDenied(l *charm.Link) {
	fmt.Println("Not Linked :(")
}

func (th *TermLinkHandler) SameAccount(l *charm.Link) {
	fmt.Println("Linked! You already linked this key btw.")
}

func (th *TermLinkHandler) Success(l *charm.Link) {
	fmt.Println("Linked!")
}

func (th *TermLinkHandler) Timeout(l *charm.Link) {
	fmt.Println("Timed out. Sorry.")
}

func (th *TermLinkHandler) Error(l *charm.Link) {
	fmt.Println("Error, something's wrong.")
}

func formatLong(s string) string {
	return indent.String(wordwrap.String("\n"+s, wrapAt), indentBy)
}

var (
	identityFile string
	cfg          *charm.Config
	cc           *charm.Client

	rootCmd = &cobra.Command{
		Use:   "charm",
		Short: "Do " + common.Keyword("Charm") + " stuff",
		Run: func(_ *cobra.Command, _ []string) {
			// Run the TUI
			if err := ui.NewProgram(cc).Start(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	bioCmd = &cobra.Command{
		Use:    "bio",
		Hidden: true,
		Short:  "",
		Long:   formatLong(""),
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			u, err := cc.Bio()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(u)
		},
	}

	idCmd = &cobra.Command{
		Use:   "id",
		Short: "Print your Charm ID",
		Long:  formatLong("Want to know your " + common.Keyword("Charm ID") + "? You’re in luck, kiddo."),
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			id, err := cc.ID()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(id)
		},
	}

	jwtCmd = &cobra.Command{
		Use:   "jwt",
		Short: "Print a JWT token",
		Long:  formatLong(common.Keyword("JWT tokens") + " are a way to authenticate to different web services that utilize your Charm account. If you’re a nerd you can use " + common.Code("jwt") + " to get one for yourself."),
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			jwt, err := cc.JWT()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s\n", jwt)
		},
	}

	keysCmd = &cobra.Command{
		Use:   "keys",
		Short: "Print linked keys",
		Long:  formatLong("Charm accounts are powered by " + common.Keyword("SSH keys") + ". This command prints all of the keys linked to your account. To remove keys use the main " + common.Code("charm") + " interface."),
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ak, err := cc.AuthorizedKeys()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(ak)
		},
	}

	linkCmd = &cobra.Command{
		Use:     "link [code]",
		Short:   "Link multiple machines to your Charm account",
		Long:    formatLong("It’s easy to " + common.Keyword("link") + " multiple machines or keys to your Charm account. Just run " + common.Code("charm link") + " on a machine connected to the account to want to link to start the process."),
		Example: indent.String("charm link\ncharm link XXXXXX", indentBy),
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			lh := &TermLinkHandler{}
			if len(args) == 0 {
				// Initialize a linking session
				p := tea.NewProgram(link.Init(cc), link.Update, link.View, link.Subscriptions)
				if err := p.Start(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				return
			}
			// Join in on a linking session
			err := cc.Link(lh, args[0])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	nameCmd = &cobra.Command{
		Use:     "name USERNAME",
		Short:   "Set your username",
		Long:    formatLong("Set a " + common.Keyword("name") + " for your account. If the name is already taken, just run it again with a different, cooler name. Basic latin letters and numbers only, and no spaces."),
		Args:    cobra.ExactArgs(1),
		Example: indent.String("charm name beatrix", indentBy),
		Run: func(cmd *cobra.Command, args []string) {
			n := args[0]
			u, err := cc.SetName(n)
			if err == charm.ErrNameTaken {
				fmt.Println("User name " + common.Code(n) + " is already taken. Try another, cooler name.")
				os.Exit(1)
			}
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("@%s ID: %s\n", u.Name, u.CharmID)
		},
	}
)

func main() {
	var err error

	// Setup Cobra
	rootCmd.PersistentFlags().StringVarP(&identityFile, "identity", "i", "", "path to identity file (that is, an ssh private key)")
	rootCmd.AddCommand(bioCmd, idCmd, jwtCmd, keysCmd, linkCmd, nameCmd)

	// Load config
	cfg, err = charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	if identityFile != "" {
		cfg.SSHKeyPath = identityFile
		cfg.ForceKey = true
	}

	// Initialize Charm client
	cc, err = charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		log.Fatal("Missing ssh key. Run `ssh-keygen` to make one or set the `CHARM_SSH_KEY_PATH` env var to your private key path.")
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Run Cobra
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
