package cmd

import (
	"github.com/Anon10214/dinkel/cmd/rerun"
)

var rerunCmd = rerun.Cmd

func init() {
	rootCmd.AddCommand(rerunCmd)

	rerunCmd.Flags().BoolVarP(&rerun.RegenerateMarkdown, "regenerate-markdown", "r", false, "Regenerate the bugreport's markdown.\nThis will overwrite the existing markdown file if present.")
}
