// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"github.com/rivo/tview"
	"github.com/spezifisch/stmps/service"
)

func makeModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

func formatSongForPlaylistEntry(entity service.SubsonicEntity) (text string) {
	if entity.Title != "" {
		text += "[::-] [white]" + tview.Escape(entity.Title)
	}
	if entity.Artist != "" {
		text += " [gray]by [white]" + tview.Escape(entity.Artist)
	}
	return
}
