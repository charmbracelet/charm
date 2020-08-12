package main

import (
	"os"

	"github.com/charmbracelet/charm/ui/common"
	"github.com/spf13/cobra"
)

func completionInstructions() string {
	return formatLong(`Charm supports shell completion for bash, zsh, fish and powershell.

` + common.Keyword("Bash") + `

To install completions:

Linux:
` + common.Code("charm completion bash > /etc/bash_completion.d/charm") + `

MacOS:
` + common.Code("charm completion bash > /usr/local/etc/bash_completion.d/charm") + `

Or, to just load for the current session:
` + common.Code("source <(charm completion bash)") + `

` + common.Keyword("Zsh") + `

If shell completion is not already enabled in your environment you will need to enable it. You can execute the following once:
` + common.Code(`echo "autoload -U compinit; compinit" >> ~/.zshrc`) + `

Then, to install completions:
` + common.Code(`charm completion zsh > "${fpath[1]}/_charm"`) + `

You will need to start a new shell for this setup to take effect.

` + common.Keyword("Fish") + `

To load completions for each session:
` + common.Code(`charm completion fish > ~/.config/fish/completions/charm.fish`) + `

Or to just load in the current session:
` + common.Code("charm completion fish | source"))
}

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "generate shell completion",
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
