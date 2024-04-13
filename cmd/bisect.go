package cmd

import (
	"github.com/Anon10214/dinkel/cmd/bisect"
)

var bisectCmd = bisect.Cmd

func init() {
	rootCmd.AddCommand(bisectCmd)
}
