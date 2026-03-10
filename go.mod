module github.com/jlarriba/o9s

go 1.25.6

require (
	github.com/gdamore/tcell/v2 v2.8.1
	github.com/gophercloud/gophercloud/v2 v2.10.0
	github.com/rivo/tview v0.0.0-20241227133733-17b7edb88c57
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/term v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/gophercloud/gophercloud/v2 => github.com/jlarriba/gophercloud/v2 v2.0.0-20260310123406-550b9bc39095
