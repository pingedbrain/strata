// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pingedbrain/strata/pkg/check"
	"github.com/pingedbrain/strata/pkg/graph"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

var (
	stTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	stDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stSel     = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	stOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	stWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	stErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	stKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	stPane    = lipgloss.NewStyle().Padding(0, 1)
	stDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type reqRow struct {
	req    *reqgraph.Requirement
	status rune // '✓' linked, '✗' uncovered, '!' stale
	edges  []graph.Edge
}

type tuiModel struct {
	rep    *check.Report
	rows   []reqRow
	sel    int
	listAt int // top of visible window
	vp     viewport.Model
	w, h   int
	ready  bool
}

func newTUIModel(rep *check.Report) tuiModel {
	rows := make([]reqRow, 0, len(rep.Graph.IDs()))
	uncovered := map[string]bool{}
	for _, r := range rep.Result.Uncovered {
		uncovered[r.ID] = true
	}
	for _, id := range rep.Graph.IDs() {
		r := rep.Graph.Requirements[id]
		edges := rep.Result.EdgesFor(id)
		status := '✓'
		if uncovered[id] {
			status = '✗'
		}
		for _, e := range edges {
			if e.Stale {
				status = '!'
				break
			}
		}
		rows = append(rows, reqRow{req: r, status: status, edges: edges})
	}
	sort.SliceStable(rows, func(i, j int) bool { // uncovered first
		return rows[i].status > rows[j].status
	})
	return tuiModel{rep: rep, rows: rows}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		vpW := m.w - m.listWidth() - 3
		m.vp = viewport.New(vpW, m.h-4)
		m.vp.SetContent(m.detail())
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "j", "down":
			m.sel = min(m.sel+1, len(m.rows)-1)
		case "k", "up":
			m.sel = max(m.sel-1, 0)
		case "ctrl+d", "pgdown":
			m.vp.HalfPageDown()
		case "ctrl+u", "pgup":
			m.vp.HalfPageUp()
		}
		// keep selection visible
		if h := m.h - 4; m.sel < m.listAt {
			m.listAt = m.sel
		} else if m.sel >= m.listAt+h {
			m.listAt = m.sel - h + 1
		}
		m.vp.SetContent(m.detail())
	}
	return m, nil
}

func (m tuiModel) listWidth() int { return 38 }

func (m tuiModel) detail() string {
	if len(m.rows) == 0 {
		return stDim.Render("no requirements")
	}
	row := m.rows[m.sel]
	var b strings.Builder
	b.WriteString(stTitle.Render(row.req.Title) + "\n")
	b.WriteString(stDim.Render(row.req.ID+"  ·  "+row.req.Source) + "\n\n")
	if row.req.Text != "" {
		b.WriteString(stDim.Render("  "+row.req.Text) + "\n\n")
	}
	for _, s := range row.req.Scenarios {
		b.WriteString("  ◇ " + stSel.Render(s.Name) + "\n")
		for _, step := range s.Steps {
			b.WriteString("      " + step + "\n")
		}
	}
	if len(row.req.Scenarios) > 0 {
		b.WriteString("\n")
	}
	if len(row.edges) == 0 {
		b.WriteString(stErr.Render("  ✗ no implementation links — uncovered requirement") + "\n")
	}
	for _, e := range row.edges {
		icon := stOK.Render("✓")
		if e.Stale {
			icon = stWarn.Render("! stale — spec changed since annotation")
		}
		sym := e.Symbol
		if sym == "" {
			sym = "(file)"
		}
		fmt.Fprintf(&b, "  %s %s:%d %s [%s]\n", icon, e.File, e.Line, sym, e.Kind)
	}
	return b.String()
}

func (m tuiModel) View() string {
	if !m.ready {
		return "loading…"
	}
	// left: requirement list
	var lb strings.Builder
	lb.WriteString(stTitle.Render(" requirements ") + "\n")
	visible := m.h - 4
	for i := m.listAt; i < len(m.rows) && i < m.listAt+visible; i++ {
		row := m.rows[i]
		icon := stOK.Render("✓")
		switch row.status {
		case '✗':
			icon = stErr.Render("✗")
		case '!':
			icon = stWarn.Render("!")
		}
		id := row.req.ID
		if w := m.listWidth() - 6; len(id) > w {
			id = id[:w-1] + "…"
		}
		line := fmt.Sprintf("%s %s", icon, id)
		if i == m.sel {
			line = stSel.Render(line)
		}
		lb.WriteString(line + "\n")
	}
	list := stPane.Width(m.listWidth()).Render(lb.String())
	divider := stDivider.Render(strings.Repeat("│\n", m.h-3))
	detail := stPane.Render(m.vp.View())

	// footer
	cov := repCoverage(m.rep)
	foot := fmt.Sprintf(" %d reqs · %s covered · %d stale · %d dangling · %d unlinked symbols    %s",
		len(m.rows), cov,
		countStale(m.rep), len(m.rep.Result.Dangling), len(m.rep.Result.Unlinked),
		stKey.Render("↑↓ navigate · pgup/dn scroll · q quit"))

	return lipgloss.JoinHorizontal(lipgloss.Top, list, divider, detail) + "\n" + stDim.Render(foot)
}

func repCoverage(rep *check.Report) string {
	impl, tot := rep.Result.Coverage()
	return fmt.Sprintf("%d/%d", impl, tot)
}

func countStale(rep *check.Report) int {
	n := 0
	for _, e := range rep.Result.Edges {
		if e.Stale {
			n++
		}
	}
	return n
}

func runTUI(args []string) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rep, err := check.Run(os.DirFS(*root), check.Options{})
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newTUIModel(rep), tea.WithAltScreen()).Run()
	return err
}
