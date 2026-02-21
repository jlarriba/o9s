package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

type Detail struct {
	view *tview.TextView
}

func NewDetail() *Detail {
	d := &Detail{
		view: tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true),
	}
	d.view.SetBorder(true).
		SetBorderColor(ColorBorder)
	return d
}

func (d *Detail) Widget() tview.Primitive {
	return d.view
}

func (d *Detail) Show(kind string, data [][2]string) {
	d.view.Clear()
	d.view.SetTitle(fmt.Sprintf(" %s ", kind))

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
