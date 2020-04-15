package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/charmbracelet/charm/ui/link"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/charmbracelet/tea"
	"github.com/mattn/go-isatty"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/spf13/cobra"
)

const (
	wrapAt   = 78
	indentBy = 2
)

func formatLong(s string) string {
	return indent.String(wordwrap.String("\n"+s, wrapAt), indentBy)
}

func printFormatted(s string) {
	fmt.Println(formatLong(s + "\n"))
}

func isTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

var (
	identityFile string
	simpleOutput bool

	cfg *charm.Config
	cc  *charm.Client

	rootCmd = &cobra.Command{
		Use:   "charm",
		Short: "Do Charm stuff",
		Long:  formatLong(fmt.Sprintf("Do %s stuff. Run without arguments for fancy mode or use the sub-commands like a pro.", common.Keyword("Charm"))),
		Run: func(_ *cobra.Command, _ []string) {
			// Run the TUI
			cfg := getCharmConfig()
			if err := ui.NewProgram(cfg).Start(); err != nil {
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
			initCharmClient()
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
			initCharmClient()
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
			initCharmClient()
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
			initCharmClient()
			if isTTY() && !simpleOutput {
				if err := keys.NewProgram(cc).Start(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			} else {
				ak, err := cc.AuthorizedKeysWithMetadata()
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				for _, v := range ak {
					fmt.Println(v.Key)
				}
			}
		},
	}

	keygenCmd = &cobra.Command{
		Use:    "keygen",
		Hidden: true,
		Short:  "Generate SSH keys",
		Long:   formatLong("Charm accounts are powered by " + common.Keyword("SSH keys") + ". This command will create them for you."),
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if isTTY() && !simpleOutput {
				err := tea.NewProgram(keygen.Init, keygen.Update, keygen.View, keygen.Subscriptions).Start()
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			} else {
				// TODO
			}
		},
	}

	linkCmd = &cobra.Command{
		Use:     "link [code]",
		Short:   "Link multiple machines to your Charm account",
		Long:    formatLong("It’s easy to " + common.Keyword("link") + " multiple machines or keys to your Charm account. Just run " + common.Code("charm link") + " on a machine connected to the account to want to link to start the process."),
		Example: indent.String("charm link\ncharm link XXXXXX", indentBy),
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			initCharmClient()
			switch len(args) {
			case 0:
				// Initialize a linking session
				p := linkgen.NewProgram(cc)
				if err := p.Start(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				return
			default:
				// Join in on a linking session
				p := link.NewProgram(cc, args[0])
				if err := p.Start(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
		},
	}

	nameCmd = &cobra.Command{
		Use:     "name [username]",
		Short:   "Username stuff",
		Long:    formatLong("Print or set your " + common.Keyword("username") + ". If the name is already taken, just run it again with a different, cooler name. Basic latin letters and numbers only, 50 characters max."),
		Args:    cobra.RangeArgs(0, 1),
		Example: indent.String("charm name\ncharm name beatrix", indentBy),
		Run: func(cmd *cobra.Command, args []string) {
			initCharmClient()
			switch len(args) {
			case 0:
				u, err := cc.Bio()
				if err != nil {
					fmt.Print(err)
					os.Exit(1)
				}
				fmt.Println(u.Name)
			default:
				n := args[0]
				if !charm.ValidateName(n) {
					msg := fmt.Sprintf("%s is invalid.\n\nUsernames must be basic latin letters, numerals, and no more than 50 characters. And no emojis, kid.\n", common.Code(n))
					fmt.Println(formatLong(msg))
					os.Exit(1)
				}
				u, err := cc.SetName(n)
				if err == charm.ErrNameTaken {
					printFormatted(fmt.Sprintf("User name %s is already taken. Try a different, cooler name.\n", common.Code(n)))
					os.Exit(1)
				}
				if err != nil {
					printFormatted(fmt.Sprintf("Welp, there’s been an error. %s", common.Subtle(err.Error())))
					fmt.Println(err)
					os.Exit(1)
				}
				printFormatted(fmt.Sprintf("OK! Your new username is %s", common.Code(u.Name)))
			}
		},
	}
)

func getCharmConfig() *charm.Config {
	cfg, err := charm.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	if identityFile != "" {
		cfg.SSHKeyPath = identityFile
		cfg.ForceKey = true
	}
	return cfg
}

func initCharmClient() *charm.Client {
	var err error

	cfg := getCharmConfig()

	cc, err = charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		log.Fatal("Missing ssh key. Run `ssh-keygen` to make one or set the `CHARM_SSH_KEY_PATH` env var to your private key path.")
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return cc
}

func main() {
	rootCmd.PersistentFlags().StringVarP(&identityFile, "identity", "i", "", "path to identity file (that is, an ssh private key)")
	keysCmd.Flags().BoolVarP(&simpleOutput, "simple", "s", false, "simple, non-interactive output (good for scripts)")
	rootCmd.AddCommand(
		bioCmd,
		idCmd,
		jwtCmd,
		keysCmd,
		keygenCmd,
		linkCmd,
		nameCmd,
	)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
