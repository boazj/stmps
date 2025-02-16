// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import "github.com/spezifisch/stmps/mpvplayer"

func (ui *Ui) SendEvent(event mpvplayer.UiEvent) {
	ui.mpvEvents <- event
}
