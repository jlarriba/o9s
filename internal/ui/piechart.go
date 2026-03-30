package ui

import (
	"fmt"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PieChart renders a terminal pie/donut chart inside a tview.Box.
type PieChart struct {
	*tview.Box
	label string
	pct   float64 // 0-100, negative means unavailable
}

// NewPieChart creates a pie chart with a label and percentage value.
func NewPieChart(label string, pct float64) *PieChart {
	return &PieChart{
		Box:   tview.NewBox(),
		label: label,
		pct:   pct,
	}
}

func (p *PieChart) Draw(screen tcell.Screen) {
	p.Box.DrawForSubclass(screen, p)
	x, y, width, height := p.GetInnerRect()

	if width < 4 || height < 3 {
		return
	}

	// Reserve 1 row at bottom for label
	chartHeight := height - 1

	// Determine circle dimensions (correct for terminal aspect ratio ~2:1)
	const aspect = 2.0
	centerX := float64(width) / 2.0
	centerY := float64(chartHeight) / 2.0
	radius := math.Min(centerX, centerY*aspect) - 1
	if radius < 1 {
		return
	}

	// Pick color based on usage level
	usedColor := tcell.ColorGreen
	if p.pct >= 90 {
		usedColor = tcell.ColorRed
	} else if p.pct >= 50 {
		usedColor = tcell.ColorOrange
	}

	usedStyle := tcell.StyleDefault.Foreground(usedColor)
	emptyStyle := tcell.StyleDefault.Foreground(tcell.ColorDimGray)
	unavailStyle := tcell.StyleDefault.Foreground(tcell.ColorDimGray)

	// Angle threshold: fraction of full circle (starting from 12 o'clock, clockwise)
	threshold := 2 * math.Pi * p.pct / 100

	// Draw the circle
	for row := 0; row < chartHeight; row++ {
		for col := 0; col < width; col++ {
			dx := float64(col) - centerX + 0.5
			dy := (float64(row) - centerY + 0.5) * aspect

			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > radius {
				continue
			}

			sx := x + col
			sy := y + row

			if p.pct < 0 {
				// Unavailable
				screen.SetContent(sx, sy, '░', nil, unavailStyle)
				continue
			}

			// Angle from 12 o'clock, clockwise
			angle := math.Atan2(dx, -dy)
			if angle < 0 {
				angle += 2 * math.Pi
			}

			if angle <= threshold {
				screen.SetContent(sx, sy, '█', nil, usedStyle)
			} else {
				screen.SetContent(sx, sy, '░', nil, emptyStyle)
			}
		}
	}

	// Draw percentage text centered in the circle
	var pctText string
	if p.pct < 0 {
		pctText = "-"
	} else {
		pctText = fmt.Sprintf("%.0f%%", p.pct)
	}
	pctX := x + int(centerX) - len(pctText)/2
	pctY := y + int(centerY)
	pctStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	for i, ch := range pctText {
		screen.SetContent(pctX+i, pctY, ch, nil, pctStyle)
	}

	// Draw label centered below the circle
	labelX := x + int(centerX) - len(p.label)/2
	labelY := y + chartHeight
	labelStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	for i, ch := range p.label {
		screen.SetContent(labelX+i, labelY, ch, nil, labelStyle)
	}
}
