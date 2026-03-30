package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

type Detail struct {
	layout *tview.Flex
	view   *tview.TextView
}

func NewDetail() *Detail {
	d := &Detail{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true),
		layout: tview.NewFlex().SetDirection(tview.FlexColumn),
	}
	d.layout.SetBorder(true).
		SetBorderColor(ColorBorder)
	d.layout.AddItem(d.view, 0, 1, true)
	return d
}

func (d *Detail) Widget() tview.Primitive {
	return d.layout
}

// Show renders a text-only detail view (used by Enter key and auto-refresh).
func (d *Detail) Show(kind string, data [][2]string) {
	d.layout.Clear()
	d.layout.AddItem(d.view, 0, 1, true)
	d.view.Clear()
	d.layout.SetTitle(fmt.Sprintf(" %s ", kind))

	// Find max key length for alignment
	maxKey := 0
	for _, kv := range data {
		if len(kv[0]) > maxKey {
			maxKey = len(kv[0])
		}
	}

	for _, kv := range data {
		fmt.Fprintf(d.view, "[indianred::b]%-*s[white::-]  %s\n", maxKey+1, kv[0]+":", kv[1])
	}
}

// ShowMetrics renders a metrics view with text on the left and pie charts on the right.
func (d *Detail) ShowMetrics(title string, data [][2]string, cpu, mem, disk float64) {
	d.layout.Clear()
	d.view.Clear()
	d.layout.SetTitle(fmt.Sprintf(" %s ", title))

	// Render text data
	maxKey := 0
	for _, kv := range data {
		if len(kv[0]) > maxKey {
			maxKey = len(kv[0])
		}
	}
	for _, kv := range data {
		fmt.Fprintf(d.view, "[indianred::b]%-*s[white::-]  %s\n", maxKey+1, kv[0]+":", kv[1])
	}

	// Build charts panel: horizontal row of 3 pie charts
	charts := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(NewPieChart("CPU", cpu), 0, 1, false).
		AddItem(NewPieChart("MEM", mem), 0, 1, false).
		AddItem(NewPieChart("DISK", disk), 0, 1, false)

	d.layout.AddItem(d.view, 0, 3, true).
		AddItem(charts, 0, 2, false)
}

// ShowLogs renders a text-only view for log output.
func (d *Detail) ShowLogs(title string, text string) {
	d.layout.Clear()
	d.layout.AddItem(d.view, 0, 1, true)
	d.view.Clear()
	d.layout.SetTitle(fmt.Sprintf(" %s ", title))
	fmt.Fprint(d.view, text)
	d.view.ScrollToEnd()
}

// ShowError renders an error message in the detail view.
func (d *Detail) ShowError(msg string) {
	d.layout.Clear()
	d.layout.AddItem(d.view, 0, 1, true)
	d.view.Clear()
	d.layout.SetTitle(" Error ")
	fmt.Fprintf(d.view, "[red]%s", msg)
}
