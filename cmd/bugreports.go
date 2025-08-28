package cmd

import (
	"github.com/Anon10214/dinkel/cmd/bugreports"
)

var bugreportsCmd = bugreports.Cmd

func init() {
	rootCmd.AddCommand(bugreportsCmd)
}
