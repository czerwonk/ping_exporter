package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type userInterface struct {
	app          *tview.Application
	grid         *tview.Grid
	table        *tview.Table
	destinations []*destination
}

var coldef = [...]struct {
	title   string
	align   int
	initVal func(*destination) string
	content func(*stat) string
}{
	{
		title:   "host",
		align:   tview.AlignLeft,
		initVal: func(d *destination) string { return d.host },
	},
	{
		title:   "address",
		align:   tview.AlignLeft,
		initVal: func(d *destination) string { return d.remote.IP.String() },
	},
	{
		title:   "sent",
		align:   tview.AlignRight,
		content: func(st *stat) string { return strconv.Itoa(st.pktSent) },
	},
	{
		title:   "loss",
		align:   tview.AlignRight,
		content: func(st *stat) string { return fmt.Sprintf("%0.1f%%", st.pktLoss*100) },
	},
	{
		title:   "last",
		align:   tview.AlignRight,
		content: func(st *stat) string { return ts(st.last) },
	},
	{
		title:   "best",
		align:   tview.AlignRight,
		content: func(st *stat) string { return ts(st.best) },
	},
	{
		title:   "worst",
		align:   tview.AlignRight,
		content: func(st *stat) string { return ts(st.worst) },
	},
	{
		title:   "mean",
		align:   tview.AlignRight,
		content: func(st *stat) string { return ts(st.mean) },
	},
	{
		title:   "stddev",
		align:   tview.AlignRight,
		content: func(st *stat) string { return ts(st.stddev) },
	},
}

func buildTUI(destinations []*destination) *userInterface {
	ui := &userInterface{
		app:          tview.NewApplication(),
		table:        tview.NewTable().SetBorders(false).SetFixed(2, 0),
		grid:         tview.NewGrid().SetRows(3, 0, 10).SetColumns(0),
		destinations: destinations,
	}

	title := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]multiping[white]   press q to exit")

	logs := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)
	log.SetFlags(log.Ltime | log.LUTC)
	log.SetOutput(logs)

	ui.grid.AddItem(title, 0, 0, 1, 1, 0, 0, false)
	ui.grid.AddItem(ui.table, 1, 0, 1, 1, 0, 0, true)
	ui.grid.AddItem(logs, 2, 0, 1, 1, 0, 0, false)

	// setup controls
	ui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			ui.app.Stop()
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				ui.app.Stop()
				return nil
			}
		}
		return event
	})

	// build header
	for col, def := range coldef {
		cell := tview.NewTableCell(def.title).SetAlign(def.align)
		if col == 2 {
			cell.SetExpansion(1)
		}
		ui.table.SetCell(0, col, cell)
	}

	// prepare data list
	for r, dst := range destinations {
		for c, def := range coldef {
			var cell *tview.TableCell
			if def.initVal != nil {
				cell = tview.NewTableCell(def.initVal(dst))
			} else {
				cell = tview.NewTableCell("n/a")
			}
			ui.table.SetCell(r+2, c, cell.SetAlign(def.align))
		}
	}

	return ui
}

func (ui *userInterface) Run() error {
	ui.app.SetRoot(ui.grid, true).SetFocus(ui.table)
	return ui.app.Run()
}

func (ui *userInterface) update(interval time.Duration) {
	time.Sleep(interval)
	for {
		for i, u := range ui.destinations {
			stats := u.compute()
			r := i + 2

			for col, def := range coldef {
				if def.content != nil {
					ui.table.GetCell(r, col).SetText(def.content(&stats))
				}
			}
		}
		ui.app.Draw()
		time.Sleep(interval)
	}
}

const tsDividend = float64(time.Millisecond) / float64(time.Nanosecond)

func ts(dur time.Duration) string {
	if 10*time.Microsecond < dur && dur < time.Second {
		return fmt.Sprintf("%0.2f", float64(dur.Nanoseconds())/tsDividend)
	}
	return dur.String()
}
