/*
Package bugreports provides the cobra-cli command for managing multiple bugreports.
*/
package bugreports

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Cmd for managing the bugreports
var Cmd = &cobra.Command{
	Use:     "bugreports",
	Aliases: []string{"bugreport", "reports", "report"},
	Short:   "Manage your bugreports",
	Long: `Manage the bugreports dinkel generated.
With this command you can easily rerun, regenerate, delete or edit the names of the generated bugreports.`,
	Run: func(cmd *cobra.Command, args []string) {
		bugreportsDir, err := cmd.Flags().GetString("bugreports")
		if err != nil {
			logrus.Errorf("Failed to get location of bugreports - %v", err)
			os.Exit(1)
		}

		p := tea.NewProgram(createInspectModel(bugreportsDir), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			logrus.Errorf("Bugreports command failed - %v", err)
			os.Exit(1)
		}
	},
}
