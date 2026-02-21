package ui

import (
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jlarriba/o9s/internal/resource"
	"github.com/rivo/tview"
)

type CommandBar struct {
	input *tview.InputField
	frame *tview.Flex
	app   *App
}

func NewCommandBar(app *App) *CommandBar {
	cb := &CommandBar{
		input: tview.NewInputField(),
		app:   app,
	}

	cb.input.SetLabel(": ").
		SetLabelColor(ColorCommand).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.ColorWhite)

	cb.frame = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cb.input, 1, 0, true)
	cb.frame.SetBorder(true).
		SetBorderColor(ColorSelected).
		SetBorderPadding(0, 0, 1, 1)

	names := resource.AllNames()
	sort.Strings(names)

	cb.input.SetAutocompleteFunc(func(current string) []string {
		if current == "" {
			return nil
		}
		current = strings.ToLower(current)
		var matches []string
		for _, name := range names {
			if strings.HasPrefix(name, current) {
				matches = append(matches, name)
			}
		}
		return matches
	})

	cb.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			text := strings.TrimSpace(cb.input.GetText())
			if text == "q" || text == "quit" {
				app.tviewApp.Stop()
				return
			}
			if text != "" {
				app.SwitchResource(text)
			}
			app.HideCommand()
		case tcell.KeyEscape:
			app.HideCommand()
		}
	})

	return cb
}

func (cb *CommandBar) Widget() tview.Primitive {
	return cb.frame
}

func (cb *CommandBar) Focus() {
	cb.input.SetText("")
	cb.app.tviewApp.SetFocus(cb.input)
}
