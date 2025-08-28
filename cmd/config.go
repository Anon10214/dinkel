package cmd

import (
	_ "embed"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Gets set by [/embed.go] because go:embed doesn't support relative filepaths
var TargetConfigTemplate string

var overwriteConfig bool

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config [output-file]",
	Short: "Generate the target config",
	Long: `Generates the target-config.yml file with the default configuration.

The output file can optionally be defined, otherwise the file will be "targets-config.yml".`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputFile := "targets-config.yml"
		if len(args) == 1 {
			outputFile = args[0]
		}

		file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_RDWR, os.FileMode(0666))
		if err != nil {
			fmt.Printf("Failed to open file %s - %v", outputFile, err)
			os.Exit(1)
		}

		if !overwriteConfig {
			contents, err := io.ReadAll(file)
			if err != nil {
				fmt.Printf("Failed to read existing file contents %s - %v", outputFile, err)
				os.Exit(1)
			}
			if len(contents) != 0 {
				fmt.Printf("Target file %s not empty and --overwrite flag not set. Aborting.\n", outputFile)
				os.Exit(1)
			}
		}

		if _, err = file.WriteString(TargetConfigTemplate); err != nil {
			fmt.Printf("Failed to write config %s - %v", outputFile, err)
			os.Exit(1)
		}

		fmt.Printf("Written config to %s!\n", outputFile)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolVar(&overwriteConfig, "overwrite", false, "Allow overwriting of existing file if contents aren't empty")
}
