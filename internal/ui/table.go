package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jlarriba/o9s/internal/client"
	"github.com/jlarriba/o9s/internal/resource"
	"github.com/rivo/tview"
)

type Table struct {
	table    *tview.Table
	frame    *tview.Flex
	app      *App
	resource resource.Resource
	rows     [][]string // cached data for ID lookup
}

func NewTable(app *App) *Table {
	t := &Table{
		table: tview.NewTable().
			SetSelectable(true, false).
			SetFixed(1, 0).
			SetBorders(false),
		app: app,
	}

	t.table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(ColorSelected))

	t.table.SetSelectedFunc(func(row, col int) {
		if row > 0 && row-1 < len(t.rows) && t.resource != nil {
			idCol := t.resource.IDColumn()
			if idCol < len(t.rows[row-1]) {
				app.ShowDetail(t.rows[row-1][idCol])
			}
		}
	})

	// Wrap table in a bordered box
	t.frame = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.table, 0, 1, true)
	t.frame.SetBorder(true).
		SetBorderColor(ColorSelected).
		SetBorderPadding(0, 0, 1, 1).
		SetTitleAlign(tview.AlignCenter)

	return t
}

func (t *Table) Widget() tview.Primitive {
	return t.frame
}

func (t *Table) setTitle(kind string, count int) {
	title := strings.ToUpper(kind[:1]) + kind[1:] + "s"
	t.frame.SetTitle(fmt.Sprintf(" [tomato::b]%s [white::b][%d][-::-] ", title, count))
}

func (t *Table) Load(ctx context.Context, res resource.Resource, c *client.OpenStack) {
	t.resource = res
	t.table.Clear()
	t.frame.SetTitle(fmt.Sprintf(" [indianred]%s [dimgray][loading...] ", strings.ToUpper(res.Kind()[:1])+res.Kind()[1:]+"s"))

	// Set header
	cols := res.Columns()
	for i, col := range cols {
		cell := tview.NewTableCell(col.Name).
			SetTextColor(ColorTitle).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		if col.Width > 0 {
			cell.SetMaxWidth(col.Width)
		}
		cell.SetExpansion(1)
		t.table.SetCell(0, i, cell)
	}

	// Show loading message
	t.table.SetCell(1, 0, tview.NewTableCell("Loading...").
		SetTextColor(ColorLabel))

	// Async load
	go func() {
		rows, err := res.List(ctx, c)
		t.app.tviewApp.QueueUpdateDraw(func() {
			// Clear loading message
			t.table.Clear()

			// Re-set header
			for i, col := range cols {
				cell := tview.NewTableCell(col.Name).
					SetTextColor(ColorTitle).
					SetSelectable(false).
					SetAttributes(tcell.AttrBold)
				if col.Width > 0 {
					cell.SetMaxWidth(col.Width)
				}
				cell.SetExpansion(1)
				t.table.SetCell(0, i, cell)
			}

			if err != nil {
				t.table.SetCell(1, 0, tview.NewTableCell("Error: "+err.Error()).
					SetTextColor(ColorError))
				t.rows = nil
				t.setTitle(res.Kind(), 0)
				return
			}

			if len(rows) == 0 {
				t.table.SetCell(1, 0, tview.NewTableCell("No resources found").
					SetTextColor(ColorLabel))
				t.rows = nil
				t.setTitle(res.Kind(), 0)
				return
			}

			t.rows = rows
			t.setTitle(res.Kind(), len(rows))
			for r, row := range rows {
				for c, val := range row {
					cell := tview.NewTableCell(val).
						SetTextColor(tcell.ColorWhite)
					// Color the status column
					if cols[c].Name == "STATUS" {
						cell.SetTextColor(StatusColor(val))
					}
					if cols[c].Width > 0 {
						cell.SetMaxWidth(cols[c].Width)
					}
					cell.SetExpansion(1)
					t.table.SetCell(r+1, c, cell)
				}
			}

			t.table.Select(1, 0)
			t.table.ScrollToBeginning()
		})
	}()
}
