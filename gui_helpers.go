// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"github.com/rivo/tview"
	"github.com/spezifisch/stmps/subsonic"
)

func makeModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

//
// func formatPlayerStatus(volume int64, position int64, duration int64) string {
// 	if position < 0 {
// 		position = 0
// 	}
//
// 	if duration < 0 {
// 		duration = 0
// 	}
//
// 	positionMin, positionSec := utils.SecondsToMinAndSec(position)
// 	durationMin, durationSec := utils.SecondsToMinAndSec(duration)
//
// 	return fmt.Sprintf("[%d%%][::b][%02d:%02d/%02d:%02d]", volume,
// 		positionMin, positionSec, durationMin, durationSec)
// }

// func formatSongForStatusBar(currentSong *mpvplayer.QueueItem) (text string) {
// 	if currentSong == nil {
// 		return
// 	}
// 	if currentSong.Title != "" {
// 		text += "[::-] [white]" + tview.Escape(currentSong.Title)
// 	}
// 	if currentSong.Artist != "" {
// 		text += " [gray]by [white]" + tview.Escape(currentSong.Artist)
// 	}
// 	return
// }

func formatSongForPlaylistEntry(entity subsonic.SubsonicEntity) (text string) {
	if entity.Title != "" {
		text += "[::-] [white]" + tview.Escape(entity.Title)
	}
	if entity.Artist != "" {
		text += " [gray]by [white]" + tview.Escape(entity.Artist)
	}
	return
}
