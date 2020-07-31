package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/charmbracelet/charm/ui/link"
	"github.com/charmbracelet/charm/ui/linkgen"
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
	randomart    bool
	forceKey     bool

	rootCmd = &cobra.Command{
		Use:   "charm",
		Short: "Do Charm stuff",
		Long:  formatLong(fmt.Sprintf("Do %s stuff. Run without arguments for fancy mode or use the sub-commands like a pro.", common.Keyword("Charm"))),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTTY() {
				cfg := getCharmConfig()

				// Log to file, if set
				if cfg.Logfile != "" {
					f, err := tea.LogToFile(cfg.Logfile, "charm")
					if err != nil {
						return err
					}
					defer f.Close()
				}

				return ui.NewProgram(cfg).Start()
			}

			return cmd.Help()
		},
	}

	bioCmd = &cobra.Command{
		Use:    "bio",
		Hidden: true,
		Short:  "",
		Long:   formatLong(""),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			u, err := cc.Bio()
			if err != nil {
				return err
			}

			fmt.Println(u)
			return nil
		},
	}

	idCmd = &cobra.Command{
		Use:   "id",
		Short: "Print your Charm ID",
		Long:  formatLong("Want to know your " + common.Keyword("Charm ID") + "? You’re in luck, kiddo."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			id, err := cc.ID()
			if err != nil {
				return err
			}

			fmt.Println(id)
			return nil
		},
	}

	jwtCmd = &cobra.Command{
		Use:   "jwt",
		Short: "Print a JWT",
		Long:  formatLong(common.Keyword("JSON Web Tokens") + " are a way to authenticate to different services that utilize your Charm account. If you’re a nerd you can use " + common.Code("jwt") + " to get one for yourself."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			jwt, err := cc.JWT()
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", jwt)
			return nil
		},
	}

	keysCmd = &cobra.Command{
		Use:   "keys",
		Short: "Browse or print linked keys",
		Long:  formatLong("Charm accounts are powered by " + common.Keyword("SSH keys") + ". This command prints all of the keys linked to your account. To remove keys use the main " + common.Code("charm") + " interface."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			if isTTY() && !simpleOutput && !randomart {

				// Log to file, if set
				cfg := getCharmConfig()
				if cfg.Logfile != "" {
					f, err := tea.LogToFile(cfg.Logfile, "charm")
					if err != nil {
						return err
					}
					defer f.Close()
				}

				return keys.NewProgram(cc).Start()

			} else {
				// Print randomart with fingerprints
				k, err := cc.AuthorizedKeysWithMetadata()
				if err != nil {
					return err
				}

				keys := k.Keys
				for i := 0; i < len(keys); i++ {
					if !randomart {
						fmt.Println(keys[i].Key)
						continue
					}
					fp, err := keys[i].FingerprintSHA256()
					if err != nil {
						fp = fmt.Sprintf("Could not generate fingerprint for key %s: %v\n\n", keys[i].Key, err)
					}
					board, err := keys[i].RandomArt()
					if err != nil {
						board = fmt.Sprintf("Could not generate randomart for key %s: %v\n\n", keys[i].Key, err)
					}
					cr := "\n\n"
					if i == len(keys)-1 {
						cr = "\n"
					}
					fmt.Printf("%s\n%s%s", fp, board, cr)
				}
				return nil
			}
		},
	}

	keygenCmd = &cobra.Command{
		Use:    "keygen",
		Hidden: true,
		Short:  "Generate SSH keys",
		Long:   formatLong("Charm accounts are powered by " + common.Keyword("SSH keys") + ". This command will create them for you."),
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTTY() && !simpleOutput {

				// Log to file if specified in the environment
				cfg := getCharmConfig()
				if cfg.Logfile != "" {
					f, err := tea.LogToFile(cfg.Logfile, "charm")
					if err != nil {
						return err
					}
					defer f.Close()
				}

				return tea.NewProgram(keygen.Init, keygen.Update, keygen.View).Start()
			} else {
				// TODO
			}

			return nil
		},
	}

	linkCmd = &cobra.Command{
		Use:     "link [code]",
		Short:   "Link multiple machines to your Charm account",
		Long:    formatLong("It’s easy to " + common.Keyword("link") + " multiple machines or keys to your Charm account. Just run " + common.Code("charm link") + " on a machine connected to the account to want to link to start the process."),
		Example: indent.String("charm link\ncharm link XXXXXX", indentBy),
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Log to file if specified in the environment
			cfg := getCharmConfig()
			if cfg.Logfile != "" {
				f, err := tea.LogToFile(cfg.Logfile, "charm")
				if err != nil {
					return err
				}
				defer f.Close()
			}

			switch len(args) {
			case 0:
				// Initialize a linking session
				p := linkgen.NewProgram(cfg)
				return p.Start()
			default:
				// Join in on a linking session
				p := link.NewProgram(cfg, args[0])
				return p.Start()
			}

		},
	}

	nameCmd = &cobra.Command{
		Use:     "name [username]",
		Short:   "Username stuff",
		Long:    formatLong("Print or set your " + common.Keyword("username") + ". If the name is already taken, just run it again with a different, cooler name. Basic latin letters and numbers only, 50 characters max."),
		Args:    cobra.RangeArgs(0, 1),
		Example: indent.String("charm name\ncharm name beatrix", indentBy),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc := initCharmClient()
			switch len(args) {
			case 0:
				u, err := cc.Bio()
				if err != nil {
					return err
				}

				fmt.Println(u.Name)
				return nil
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
					return err
				}

				printFormatted(fmt.Sprintf("OK! Your new username is %s", common.Code(u.Name)))
				return nil
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
	if forceKey {
		cfg.ForceKey = true
	}
	return cfg
}

func initCharmClient() *charm.Client {
	cfg := getCharmConfig()
	cc, err := charm.NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		printFormatted("We were’t able to authenticate via SSH, which means there’s likely a problem with your key.\n\nYou can generate SSH keys by running " + common.Code("charm keygen") + ". You can also set the environment variable " + common.Code("CHARM_SSH_KEY_PATH") + " to point to a specific private key, or use " + common.Code("-i") + "specifify a location.")
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return cc
}

func main() {
	// General flags
	rootCmd.PersistentFlags().StringVarP(&identityFile, "identity", "i", "", "path to identity file (that is, an ssh private key)")
	rootCmd.Flags().BoolVarP(&forceKey, "force-key", "f", false, "for the use of the SSH key on disk (that is, ignore ssh-agent)")

	// Keys flags
	keysCmd.Flags().BoolVarP(&simpleOutput, "simple", "s", false, "simple, non-interactive output (good for scripts)")
	keysCmd.Flags().BoolVarP(&randomart, "randomart", "r", false, "print SSH 5.1 randomart for each key (the Drunken Bishop algorithm)")

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
