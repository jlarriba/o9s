package ui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
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
		case tcell.KeyCtrlD:
			a.deleteSelected()
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
			case 'j':
				a.startServer()
				return nil
			case 'l':
				a.stopServer()
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

func (a *App) getSelectedID() string {
	if a.currentRes == nil || a.table.rows == nil {
		return ""
	}
	row, _ := a.table.table.GetSelection()
	if row < 1 || row-1 >= len(a.table.rows) {
		return ""
	}
	idCol := a.currentRes.IDColumn()
	if idCol >= len(a.table.rows[row-1]) {
		return ""
	}
	return a.table.rows[row-1][idCol]
}

func (a *App) getSelectedName() string {
	if a.table.rows == nil {
		return ""
	}
	row, _ := a.table.table.GetSelection()
	if row < 1 || row-1 >= len(a.table.rows) {
		return ""
	}
	return a.table.rows[row-1][0]
}

func (a *App) confirm(message string, onYes func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(idx int, label string) {
			a.pages.RemovePage("confirm")
			if label == "Yes" {
				onYes()
			}
			a.tviewApp.SetFocus(a.table.table)
		})
	modal.SetBackgroundColor(tcell.ColorDarkRed)
	a.pages.AddPage("confirm", modal, true, true)
}

func (a *App) deleteSelected() {
	id := a.getSelectedID()
	name := a.getSelectedName()
	if id == "" {
		return
	}
	kind := a.currentRes.Kind()
	a.confirm(fmt.Sprintf("Delete %s %q?", kind, name), func() {
		go func() {
			err := a.currentRes.Delete(context.Background(), a.osClient, id)
			a.tviewApp.QueueUpdateDraw(func() {
				if err != nil {
					a.showError(fmt.Sprintf("delete failed: %s", err))
					return
				}
				a.Reload()
			})
		}()
	})
}

func (a *App) startServer() {
	if a.currentRes == nil || a.currentRes.Kind() != "server" {
		return
	}
	id := a.getSelectedID()
	name := a.getSelectedName()
	if id == "" {
		return
	}
	a.confirm(fmt.Sprintf("Start server %q?", name), func() {
		go func() {
			computeClient, err := a.osClient.Compute()
			if err != nil {
				a.tviewApp.QueueUpdateDraw(func() { a.showError(err.Error()) })
				return
			}
			err = servers.Start(context.Background(), computeClient, id).ExtractErr()
			a.tviewApp.QueueUpdateDraw(func() {
				if err != nil {
					a.showError(fmt.Sprintf("start failed: %s", err))
					return
				}
				a.Reload()
			})
		}()
	})
}

func (a *App) stopServer() {
	if a.currentRes == nil || a.currentRes.Kind() != "server" {
		return
	}
	id := a.getSelectedID()
	name := a.getSelectedName()
	if id == "" {
		return
	}
	a.confirm(fmt.Sprintf("Stop server %q?", name), func() {
		go func() {
			computeClient, err := a.osClient.Compute()
			if err != nil {
				a.tviewApp.QueueUpdateDraw(func() { a.showError(err.Error()) })
				return
			}
			err = servers.Stop(context.Background(), computeClient, id).ExtractErr()
			a.tviewApp.QueueUpdateDraw(func() {
				if err != nil {
					a.showError(fmt.Sprintf("stop failed: %s", err))
					return
				}
				a.Reload()
			})
		}()
	})
}
