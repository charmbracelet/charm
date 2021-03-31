package main

import (
	"os"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

func completionInstructions() string {
	return formatLong(`Charm supports ` + common.Keyword("shell completion") + ` for bash, zsh, fish and powershell.

` + common.Keyword("Bash") + `

To install completions:

Linux (as root):
$ charm completion bash > /etc/bash_completion.d/charm

MacOS:
$ charm completion bash > /usr/local/etc/bash_completion.d/charm

Note that on macOS you'll need to have bash completion installed. The easiest
way to do this is with Homewbrew. For more info run: brew info bash-completion.

Or, to just load Charm completion for the current session:
$ source <(charm completion bash)

` + common.Keyword("Zsh") + `

If shell completion is not already enabled in your environment you will need to enable it. You can execute the following once:
$ echo "autoload -U compinit; compinit" >> ~/.zshrc

Then, to install completions:
$ charm completion zsh > "${fpath[1]}/_charm"

You will need to start a new shell for this setup to take effect.

` + common.Keyword("Fish") + `

To load completions for each session:
$ charm completion fish > ~/.config/fish/completions/charm.fish

Or to just load in the current session:
$ charm completion fish | source`)
}

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "Generate shell completion",
	Long:                  completionInstructions(),
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}
