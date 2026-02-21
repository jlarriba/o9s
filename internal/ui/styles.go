package ui

import "github.com/gdamore/tcell/v2"

var (
	ColorTitle      = tcell.ColorSteelBlue
	ColorHeader     = tcell.ColorWhite
	ColorSelected   = tcell.ColorDodgerBlue
	ColorBorder     = tcell.ColorDimGray
	ColorLogo       = tcell.ColorSteelBlue
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

const Logo = `  ___  ___  __
 / _ \/ _ \/ _/
 \___/\_, /\__/
     /___/`
