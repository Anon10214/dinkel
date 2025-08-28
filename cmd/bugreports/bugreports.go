package bugreports

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Anon10214/dinkel/cmd/config"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wrap"
	"github.com/sirupsen/logrus"
)

type inspectState int

const (
	// Looking at the list of reports
	main inspectState = iota
	// Renaming a report
	renaming
	// Looking at MD
	inspecting
	// Regenerating or rerunning a bugreport
	running
)

// The model when inspecting all bug reports
type inspectModel struct {
	state               inspectState
	width               int
	height              int
	bugreportsDirectory string
	table               table.Model
	renameInput         textinput.Model
	viewport            viewport.Model
	help                help.Model
	keys                inspectModelKeyMap
	// Path to the file being inspected
	inspectPath string
}

type inspectModelKeyMap struct {
	Refresh    key.Binding
	Inspect    key.Binding
	Rename     key.Binding
	Rerun      key.Binding
	RegenMd    key.Binding
	Regenerate key.Binding
	Reduce     key.Binding
	Delete     key.Binding
	Quit       key.Binding
}

func (k inspectModelKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Refresh, k.Inspect, k.Rename, k.Rerun, k.RegenMd, k.Regenerate, k.Reduce, k.Delete}
}

func (k inspectModelKeyMap) FullHelp() [][]key.Binding {
	return nil
}

func createInspectModel(bugreportsDirectory string) inspectModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Name", Width: 30},
			{Title: "Target", Width: 20},
			{Title: "Found", Width: 20},
			{Title: "Strategy", Width: 10},
			{Title: "Has MD", Width: 6},
			{Title: "Commit", Width: 7},
		}),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	t.SetStyles(s)

	model := inspectModel{
		bugreportsDirectory: bugreportsDirectory,
		table:               t,
		help:                help.New(),
		keys: inspectModelKeyMap{
			Refresh: key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "refresh"),
			),
			Inspect: key.NewBinding(
				key.WithKeys("i"),
				key.WithHelp("i", "inspect MD"),
			),
			Rename: key.NewBinding(
				key.WithKeys("n", "f2"),
				key.WithHelp("n/f2", "rename"),
			),
			Rerun: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "rerun"),
			),
			RegenMd: key.NewBinding(
				key.WithKeys("m"),
				key.WithHelp("m", "regenerate MD"),
			),
			Regenerate: key.NewBinding(
				key.WithKeys("g"),
				key.WithHelp("g", "regenerate"),
			),
			Reduce: key.NewBinding(
				key.WithKeys("d"),
				key.WithHelp("d", "reduce"),
			),
			Delete: key.NewBinding(
				key.WithKeys("x"),
				key.WithHelp("x", "delete"),
			),
			Quit: key.NewBinding(
				key.WithKeys("q", "esc", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
		},
	}

	model = model.refreshBugreports()

	return model
}

func (m inspectModel) refreshBugreports() inspectModel {
	reports, err := os.ReadDir(m.bugreportsDirectory)
	if err != nil {
		logrus.Errorf("Couldn't read in bug reports - %v", err)
		os.Exit(1)
	}

	var reportNames []table.Row
	for _, report := range reports {
		if report.Type().IsRegular() {
			name := strings.Split(report.Name(), ".")
			if len(name) == 2 && name[1] == "yml" {
				bugreport, err := config.ReadBugreport(path.Join(m.bugreportsDirectory, report.Name()))
				if err != nil {
					// Just display the error as the report name
					reportNames = append(reportNames, []string{err.Error()})
					continue
				}
				row := []string{name[0], bugreport.Target, bugreport.TimeFound[:19], bugreport.StrategyNum.ToString()}
				// Check if the corresponding MD exists
				if _, err := os.Stat(path.Join(m.bugreportsDirectory, name[0]+".md")); err == nil {
					row = append(row, "✔")
				} else {
					row = append(row, "❌")
				}
				if bugreport.OffendingCommit == "" {
					row = append(row, "❌")
				} else if len(bugreport.OffendingCommit) < 7 {
					row = append(row, bugreport.OffendingCommit)
				} else {
					row = append(row, bugreport.OffendingCommit[:7])
				}
				reportNames = append(reportNames, row)
			}
		}
	}

	m.table.SetRows(reportNames)

	return m
}

func (m inspectModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inspectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {

	case main:
		switch msg := msg.(type) {

		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.help.Width = msg.Width
			m.table.SetHeight(m.height - 2)

		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keys.Refresh):
				return m.refreshBugreports(), nil
			case key.Matches(msg, m.keys.Inspect):
				m.inspectPath = path.Join(m.bugreportsDirectory, m.table.SelectedRow()[0]+".md")

				buf, err := os.ReadFile(m.inspectPath)
				if err != nil {
					fmt.Println(err)
					break
				}

				m.viewport = viewport.New(m.width, m.height-2)
				m.viewport.SetContent(wrap.String(string(buf), m.width))

				m.state = inspecting

				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)

				return m, cmd
			case key.Matches(msg, m.keys.Rename):
				m.state = renaming

				ti := textinput.New()
				ti.SetValue(m.table.SelectedRow()[0])
				ti.Prompt = "$ "
				ti.Focus()
				ti.CharLimit = 30
				m.renameInput = ti

				return m.Update(nil)
			case key.Matches(msg, m.keys.Rerun):
				bugreportPath := path.Join(m.bugreportsDirectory, m.table.SelectedRow()[0]+".yml")

				executable, err := os.Executable()
				if err != nil {
					break
				}

				m.state = running

				return m, tea.Batch(tea.ClearScreen, tea.ExecProcess(exec.Command("/bin/sh", "-c", strings.Join([]string{executable, "rerun", bugreportPath, ";", "read", "-n1"}, " ")), nil))
			case key.Matches(msg, m.keys.RegenMd):
				bugreportPath := path.Join(m.bugreportsDirectory, m.table.SelectedRow()[0]+".yml")

				executable, err := os.Executable()
				if err != nil {
					break
				}

				m.state = running

				return m, tea.Batch(tea.ClearScreen, tea.ExecProcess(exec.Command("/bin/sh", "-c", strings.Join([]string{executable, "rerun", bugreportPath, "-r", ";", "read", "-n1"}, " ")), nil))
			case key.Matches(msg, m.keys.Regenerate):
				bugreportPath := path.Join(m.bugreportsDirectory, m.table.SelectedRow()[0]+".yml")

				executable, err := os.Executable()
				if err != nil {
					break
				}

				m.state = running

				return m, tea.Batch(tea.ClearScreen, tea.ExecProcess(exec.Command("/bin/sh", "-c", strings.Join([]string{executable, "regenerate", bugreportPath, ";", "read", "-n1"}, " ")), nil))
			case key.Matches(msg, m.keys.Reduce):
				bugreportPath := path.Join(m.bugreportsDirectory, m.table.SelectedRow()[0]+".yml")

				executable, err := os.Executable()
				if err != nil {
					break
				}

				m.state = running

				return m, tea.Batch(tea.ClearScreen, tea.ExecProcess(exec.Command("/bin/sh", "-c", strings.Join([]string{executable, "reduce", bugreportPath, ";", "read", "-n1"}, " ")), nil))
			case key.Matches(msg, m.keys.Delete):
				row := m.table.SelectedRow()

				if m.table.Cursor() == len(m.table.Rows())-1 {
					m.table.SetCursor(m.table.Cursor() - 1)
				}

				os.Remove(path.Join(m.bugreportsDirectory, row[0]+".yml"))
				os.Remove(path.Join(m.bugreportsDirectory, row[0]+".md"))
				m = m.refreshBugreports()

				return m, nil
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			}
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

	case renaming:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				m.state = main

				old := path.Join(m.bugreportsDirectory, m.table.SelectedRow()[0])
				new := path.Join(m.bugreportsDirectory, m.renameInput.Value())

				os.Rename(old+".yml", new+".yml")
				os.Rename(old+".md", new+".md")

				m = m.refreshBugreports()

				return m, nil
			case tea.KeyEscape:
				m.state = main
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd

	case inspecting:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEscape:
				m.state = main
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case running:
		m.state = main
	}
	return m, nil
}

func (m inspectModel) View() string {
	switch m.state {
	case main, renaming:
		reportsView := m.table.View()

		helpView := m.help.View(m.keys)

		var renameView string
		if m.state == renaming {
			renameView = m.renameInput.View()
		}

		height := m.height - strings.Count(reportsView, "\n") - strings.Count(renameView, "\n") - strings.Count(helpView, "\n") - 2
		if height < 0 {
			height = 0
		}

		return reportsView + "\n" + renameView + strings.Repeat("\n", height) + helpView
	case inspecting:
		return m.viewport.View()
	}
	return ""
}
