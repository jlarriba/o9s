package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	// Use single-line borders even when focused (default focus uses double lines)
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
}

var (
	ColorTitle      = tcell.ColorIndianRed
	ColorHeader     = tcell.ColorWhite
	ColorSelected   = tcell.NewRGBColor(100, 30, 30)
	ColorBorder     = tcell.ColorDimGray
	ColorLogo       = tcell.ColorIndianRed
	ColorLabel      = tcell.ColorGray
	ColorValue      = tcell.ColorWhite
	ColorCommand    = tcell.ColorYellow
	ColorError      = tcell.ColorRed
	ColorActiveProj = tcell.ColorGreen
)

func StatusColor(status string) tcell.Color {
	switch status {
	case "ACTIVE", "active", "UP", "ENABLED":
		return tcell.ColorGreen
	case "BUILD", "building", "CREATING", "creating":
		return tcell.ColorYellow
	case "ERROR", "error", "FAULT":
		return tcell.ColorRed
	case "SHUTOFF", "DOWN", "DISABLED", "shutoff":
		return tcell.ColorGray
	case "PAUSED", "SUSPENDED", "VERIFY_RESIZE":
		return tcell.ColorOrange
	default:
		return tcell.ColorWhite
	}
}

const Logo = `
      ________
  ____/   __   \______
 /  _ \____    /  ___/
(  <_> ) /    /\___ \
 \____/ /____//____  >
                   \/ `
