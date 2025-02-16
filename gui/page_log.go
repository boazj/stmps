// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"time"

	"github.com/rivo/tview"
)

type LogPage struct {
	Root *tview.Flex

	logList *tview.List

	// external refs
	ui *Ui
}

func (ui *Ui) createLogPage() *LogPage {
	logPage := LogPage{
		ui: ui,
	}

	logPage.logList = tview.NewList().ShowSecondaryText(false)

	logPage.Root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(logPage.logList, 0, 1, true)

	return &logPage
}

func (l *LogPage) Print(line string) {
	l.ui.app.QueueUpdateDraw(func() {
		line := time.Now().Local().Format("(15:04:05) ") + line
		l.logList.InsertItem(0, line, "", 0, nil)

		// Make sure the log list doesn't grow infinitely
		for l.logList.GetItemCount() > 100 {
			l.logList.RemoveItem(-1)
		}
	})
}
