package cmd

import (
	"github.com/Anon10214/dinkel/cmd/rerun"
)

var rerunCmd = rerun.Cmd

func init() {
	rootCmd.AddCommand(rerunCmd)
}
