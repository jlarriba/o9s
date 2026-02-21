package ui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/jlarriba/o9s/internal/client"
	"github.com/jlarriba/o9s/internal/resource"
	"github.com/rivo/tview"
)

type App struct {
	tviewApp    *tview.Application
	osClient    *client.OpenStack
	header      *Header
	table       *Table
	detail      *Detail
	commandBar  *CommandBar
	pages      *tview.Pages
	layout     *tview.Flex
	currentRes resource.Resource
	quotas     client.QuotaInfo
	showingCmd bool
}

func NewApp(c *client.OpenStack) *App {
	app := &App{
		tviewApp: tview.NewApplication(),
		osClient: c,
	}

	app.header = NewHeader()
	app.table = NewTable(app)
	app.detail = NewDetail()
	app.commandBar = NewCommandBar(app)

	// Pages for switching between table and detail
	app.pages = tview.NewPages().
		AddPage("table", app.table.Widget(), true, true).
		AddPage("detail", app.detail.Widget(), true, false)

	// Header height: max of 8 (4 info + 1 blank + 3 quota bars) or project count, plus 1 for spacing
	headerHeight := len(c.Projects)
	if headerHeight > 10 {
		headerHeight = 10
	}
	if headerHeight < 9 {
		headerHeight = 9
	}
	headerHeight++ // extra row for spacing

	// Main layout: header, then command bar (hidden), then results box
	app.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(app.header.Widget(), headerHeight, 0, false).
		AddItem(app.commandBar.Widget(), 0, 0, false). // height 0 = hidden
		AddItem(app.pages, 0, 1, true)

	app.setupKeys()

	return app
}

func (a *App) Run() error {
	a.tviewApp.SetRoot(a.layout, true)
	// Load default resource after the event loop starts
	go func() {
		a.tviewApp.QueueUpdateDraw(func() {
			a.SwitchResource("server")
		})
	}()
	return a.tviewApp.Run()
}

func (a *App) setupKeys() {
	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Don't capture keys when command bar is focused
		if a.showingCmd {
			return event
		}

		switch event.Key() {
		case tcell.KeyEscape:
			if name, _ := a.pages.GetFrontPage(); name == "detail" {
				a.pages.SwitchToPage("table")
				a.tviewApp.SetFocus(a.table.table)
				return nil
			}
		case tcell.KeyF1:
			a.SwitchResource("server")
			return nil
		case tcell.KeyF2:
			a.SwitchResource("network")
			return nil
		case tcell.KeyF3:
			a.SwitchResource("subnet")
			return nil
		case tcell.KeyF4:
			a.SwitchResource("volume")
			return nil
		case tcell.KeyF5:
			a.SwitchResource("router")
			return nil
		case tcell.KeyCtrlC:
			a.tviewApp.Stop()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case ':':
				a.ShowCommand()
				return nil
			case 'q':
				if name, _ := a.pages.GetFrontPage(); name == "detail" {
					a.pages.SwitchToPage("table")
					a.tviewApp.SetFocus(a.table.table)
					return nil
				}
				a.tviewApp.Stop()
				return nil
			case 'r':
				a.Reload()
				return nil
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				idx := int(event.Rune() - '0')
				a.switchProjectByIndex(idx)
				return nil
			}
		}
		return event
	})
}

func (a *App) SwitchResource(name string) {
	res, err := resource.Resolve(name)
	if err != nil {
		a.showError(err.Error())
		return
	}
	a.currentRes = res
	a.refreshQuotas()
	a.header.Update(a.osClient, res.Kind(), a.quotas)
	a.pages.SwitchToPage("table")
	a.table.Load(context.Background(), res, a.osClient)
	a.tviewApp.SetFocus(a.table.table)
}

func (a *App) refreshQuotas() {
	a.quotas = a.osClient.FetchQuotas(context.Background())
}

func (a *App) Reload() {
	if a.currentRes != nil {
		a.table.Load(context.Background(), a.currentRes, a.osClient)
	}
}

func (a *App) ShowDetail(id string) {
	if a.currentRes == nil {
		return
	}
	go func() {
		data, err := a.currentRes.Show(context.Background(), a.osClient, id)
		a.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				a.detail.view.Clear()
				a.detail.view.SetTitle(" Error ")
				fmt.Fprintf(a.detail.view, "[red]%s", err.Error())
			} else {
				a.detail.Show(a.currentRes.Kind(), data)
			}
			a.pages.SwitchToPage("detail")
			a.tviewApp.SetFocus(a.detail.view)
		})
	}()
}

func (a *App) ShowCommand() {
	a.showingCmd = true
	a.layout.ResizeItem(a.commandBar.Widget(), 3, 0) // 1 input + 2 border
	a.commandBar.Focus()
}

func (a *App) HideCommand() {
	a.showingCmd = false
	a.layout.ResizeItem(a.commandBar.Widget(), 0, 0)
	a.tviewApp.SetFocus(a.table.table)
}

func (a *App) switchProjectByIndex(idx int) {
	if idx >= len(a.osClient.Projects) {
		return
	}
	proj := a.osClient.Projects[idx]
	go func() {
		err := a.osClient.SwitchProject(context.Background(), proj.ID, proj.Name)
		a.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				a.showError(fmt.Sprintf("switch project: %s", err))
				return
			}
			a.refreshQuotas()
			a.header.Update(a.osClient, a.currentRes.Kind(), a.quotas)
			a.Reload()
		})
	}()
}

// showError displays an error in the table area.
// Safe to call from any context (inside or outside QueueUpdateDraw).
func (a *App) showError(msg string) {
	a.table.table.Clear()
	a.table.table.SetCell(0, 0, tview.NewTableCell("Error").
		SetTextColor(ColorError).
		SetAttributes(tcell.AttrBold))
	a.table.table.SetCell(1, 0, tview.NewTableCell(msg).
		SetTextColor(ColorError))
}
