package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/jlarriba/o9s/internal/resource"
	"github.com/rivo/tview"
)

type Detail struct {
	layout   *tview.Flex
	view     *tview.TextView
	relTable *tview.Table
	related  []resource.RelatedResource
	onSelect func(resource.RelatedResource)
}

func NewDetail() *Detail {
	d := &Detail{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true),
		layout:   tview.NewFlex().SetDirection(tview.FlexColumn),
		relTable: tview.NewTable().SetSelectable(true, false).SetFixed(1, 0),
	}
	d.layout.SetBorder(true).
		SetBorderColor(ColorBorder)
	d.layout.AddItem(d.view, 0, 1, true)

	d.relTable.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(ColorSelected))
	d.relTable.SetBorder(true).
		SetBorderColor(ColorBorder).
		SetTitle(" [darkorange::b]Related Resources [-::-]")

	d.relTable.SetSelectedFunc(func(row, col int) {
		if row > 0 && row-1 < len(d.related) && d.onSelect != nil {
			d.onSelect(d.related[row-1])
		}
	})

	return d
}

func (d *Detail) Widget() tview.Primitive {
	return d.layout
}

// SetOnRelatedSelect sets the callback for when Enter is pressed on a related resource.
func (d *Detail) SetOnRelatedSelect(fn func(resource.RelatedResource)) {
	d.onSelect = fn
}

// FocusableWidgets returns the text view and the related table (if visible).
func (d *Detail) FocusableWidgets() (text *tview.TextView, relTable *tview.Table) {
	if len(d.related) > 0 {
		return d.view, d.relTable
	}
	return d.view, nil
}

// Show renders a text-only detail view (used by Enter key and auto-refresh).
func (d *Detail) Show(kind string, data [][2]string) {
	d.layout.Clear()
	d.layout.SetDirection(tview.FlexColumn)
	d.layout.AddItem(d.view, 0, 1, true)
	d.view.Clear()
	d.related = nil
	d.layout.SetTitle(fmt.Sprintf(" %s ", kind))

	d.renderText(data)
}

// SetRelated adds a related resources table below the text view.
func (d *Detail) SetRelated(related []resource.RelatedResource) {
	d.related = related
	if len(related) == 0 {
		return
	}

	// Rebuild layout as vertical stack: text on top, related table on bottom
	d.layout.Clear()
	d.layout.SetDirection(tview.FlexRow)

	d.relTable.Clear()
	d.relTable.SetCell(0, 0, tview.NewTableCell("TYPE").
		SetTextColor(ColorTitle).SetSelectable(false).SetAttributes(tcell.AttrBold).SetExpansion(1))
	d.relTable.SetCell(0, 1, tview.NewTableCell("NAME").
		SetTextColor(ColorTitle).SetSelectable(false).SetAttributes(tcell.AttrBold).SetExpansion(1))

	for i, rel := range related {
		d.relTable.SetCell(i+1, 0, tview.NewTableCell(rel.Kind).
			SetTextColor(tcell.ColorGray).SetExpansion(1))
		d.relTable.SetCell(i+1, 1, tview.NewTableCell(rel.DisplayName).
			SetTextColor(tcell.ColorWhite).SetExpansion(1))
	}
	d.relTable.Select(1, 0)
	d.relTable.ScrollToBeginning()

	relHeight := len(related) + 3 // rows + header + borders
	if relHeight < 5 {
		relHeight = 5
	}

	d.layout.AddItem(d.view, 0, 1, false).
		AddItem(d.relTable, relHeight, 0, true)
}

// ShowMetrics renders a metrics view with text on the left and pie charts on the right.
func (d *Detail) ShowMetrics(title string, data [][2]string, cpu, mem, disk float64) {
	d.layout.Clear()
	d.layout.SetDirection(tview.FlexColumn)
	d.view.Clear()
	d.related = nil
	d.layout.SetTitle(fmt.Sprintf(" %s ", title))

	d.renderText(data)

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
	d.layout.SetDirection(tview.FlexColumn)
	d.layout.AddItem(d.view, 0, 1, true)
	d.view.Clear()
	d.related = nil
	d.layout.SetTitle(fmt.Sprintf(" %s ", title))
	fmt.Fprint(d.view, text)
	d.view.ScrollToEnd()
}

// ShowError renders an error message in the detail view.
func (d *Detail) ShowError(msg string) {
	d.layout.Clear()
	d.layout.SetDirection(tview.FlexColumn)
	d.layout.AddItem(d.view, 0, 1, true)
	d.view.Clear()
	d.related = nil
	d.layout.SetTitle(" Error ")
	fmt.Fprintf(d.view, "[red]%s", msg)
}

func (d *Detail) renderText(data [][2]string) {
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
