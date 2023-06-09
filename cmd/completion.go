package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func completionInstructions() string {
	return paragraph(`Charm supports ` + keyword("shell completion") + ` for bash, zsh, fish and powershell.

` + keyword("Bash") + `

To install completions:

Linux (as root):
$ charm completion bash > /etc/bash_completion.d/charm

MacOS:
$ charm completion bash > /usr/local/etc/bash_completion.d/charm

Note that on macOS you'll need to have bash completion installed. The easiest
way to do this is with Homewbrew. For more info run: brew info bash-completion.

Or, to just load Charm completion for the current session:
$ source <(charm completion bash)

` + keyword("Zsh") + `

If shell completion is not already enabled in your environment you will need to enable it. You can execute the following once:
$ echo "autoload -U compinit; compinit" >> ~/.zshrc

Then, to install completions:
$ charm completion zsh > "${fpath[1]}/_charm"

You will need to start a new shell for this setup to take effect.

` + keyword("Fish") + `

To load completions for each session:
$ charm completion fish > ~/.config/fish/completions/charm.fish

Or to just load in the current session:
$ charm completion fish | source`)
}

// CompletionCmd is the cobra.Command to generate shell completion.
var CompletionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "Generate shell completion",
	Long:                  completionInstructions(),
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout) // nolint: errcheck
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout) // nolint: errcheck
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true) // nolint: errcheck
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout) // nolint: errcheck
		}
	},
}
