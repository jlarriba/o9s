package ui

import (
	"fmt"
	"strings"

	"github.com/jlarriba/o9s/internal/client"
	"github.com/rivo/tview"
)

var shortcuts = [][2]string{
	{"F1", "server"},
	{"F2", "network"},
	{"F3", "subnet"},
	{"F4", "volume"},
	{"F5", "router"},
}

var actions = [][2]string{
	{"Ctrl-d", "delete"},
	{"m", "start VM"},
	{"n", "stop VM"},
	{"b", "reboot VM"},
	{"l", "logs VM"},
	{"r", "reload"},
}

type Header struct {
	layout   *tview.Flex
	info     *tview.TextView
	projects *tview.TextView
	quicknav *tview.TextView
	actionsv *tview.TextView
	logo     *tview.TextView
}

func NewHeader() *Header {
	h := &Header{
		info:     tview.NewTextView().SetDynamicColors(true),
		projects: tview.NewTextView().SetDynamicColors(true).SetScrollable(true),
		quicknav: tview.NewTextView().SetDynamicColors(true),
		actionsv: tview.NewTextView().SetDynamicColors(true),
		logo:     tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight),
	}

	h.layout = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(h.info, 0, 1, false).
		AddItem(h.projects, 0, 1, false).
		AddItem(h.quicknav, 0, 1, false).
		AddItem(h.actionsv, 0, 1, false).
		AddItem(h.logo, 18, 0, false)

	// Set logo
	h.logo.SetText(fmt.Sprintf("[indianred]%s", Logo))

	return h
}

func (h *Header) Widget() tview.Primitive {
	return h.layout
}

func (h *Header) Update(c *client.OpenStack, currentResource string, quotas client.QuotaInfo) {
	h.info.Clear()
	fmt.Fprintf(h.info, "[white::b]Cloud:[white::-]    %s\n", c.CloudName)
	fmt.Fprintf(h.info, "[white::b]User:[white::-]     %s\n", c.UserName)
	fmt.Fprintf(h.info, "[white::b]Region:[white::-]   %s\n", c.Region)
	// Quota bars
	fmt.Fprintf(h.info, "\n")
	fmt.Fprintf(h.info, "%s\n", quotaBar("VCPUs", quotas.VCPUs, "", 20))
	fmt.Fprintf(h.info, "%s\n", quotaBar("RAM", quotas.RAM, "MB", 20))
	fmt.Fprintf(h.info, "%s\n", quotaBar("Volumes", quotas.Volumes, "", 20))
	fmt.Fprintf(h.info, "%s", quotaBar("Storage", quotas.Storage, "GB", 20))

	h.projects.Clear()
	for i, p := range c.Projects {
		if i > 9 {
			break
		}
		if p.ID == c.ProjectID {
			fmt.Fprintf(h.projects, "[green::b]<%d> %s[-::-]\n", i, p.Name)
		} else {
			fmt.Fprintf(h.projects, "[white]<%d> [gray]%s[-]\n", i, p.Name)
		}
	}

	h.quicknav.Clear()
	for _, s := range shortcuts {
		if s[1] == currentResource {
			fmt.Fprintf(h.quicknav, "[indianred::b]<%s>[white::b] %s[-::-]\n", s[0], s[1])
		} else {
			fmt.Fprintf(h.quicknav, "[dimgray]<%s> [gray]%s[-]\n", s[0], s[1])
		}
	}

	h.actionsv.Clear()
	for _, a := range actions {
		fmt.Fprintf(h.actionsv, "[darkorange::b]<%s> [gray]%s[-::-]\n", a[0], a[1])
	}
}

func quotaBar(label string, q client.QuotaUsage, unit string, width int) string {
	if q.Limit <= 0 {
		return fmt.Sprintf("[white::b]%-8s[dimgray] n/a", label)
	}

	ratio := float64(q.InUse) / float64(q.Limit)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))

	// Color based on usage
	barColor := "green"
	if ratio > 0.8 {
		barColor = "red"
	} else if ratio > 0.6 {
		barColor = "yellow"
	}

	bar := fmt.Sprintf("[%s]%s[dimgray]%s",
		barColor,
		strings.Repeat("█", filled),
		strings.Repeat("░", width-filled))

	if unit != "" {
		return fmt.Sprintf("[white::b]%-8s[-::-]%s [white]%d[dimgray]/%d %s", label, bar, q.InUse, q.Limit, unit)
	}
	return fmt.Sprintf("[white::b]%-8s[-::-]%s [white]%d[dimgray]/%d", label, bar, q.InUse, q.Limit)
}
