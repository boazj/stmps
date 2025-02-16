// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"strings"

	"github.com/rivo/tview"
	"github.com/spezifisch/stmps/consts"
)

type HelpWidget struct {
	Root *tview.Flex

	helpBook                *tview.Flex
	leftColumn, rightColumn *tview.TextView

	// visible reflects whether the modal is shown
	visible bool

	// external references
	ui *Ui
}

func (ui *Ui) createHelpWidget() (m *HelpWidget) {
	m = &HelpWidget{
		ui: ui,
	}

	// two help columns side by side
	m.leftColumn = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)
	m.rightColumn = tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)
	m.helpBook = tview.NewFlex().
		SetDirection(tview.FlexColumn)

	m.Root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.helpBook, 0, 1, false)

	m.Root.Box.SetBorder(true).SetTitle(" Help ")

	return
}

func (h *HelpWidget) RenderHelp(context string) {
	leftText := "[::b]Playback[::-]\n" + tview.Escape(strings.TrimSpace(consts.HelpPlayback))
	h.leftColumn.SetText(leftText)

	rightText := ""
	switch context {
	case PageBrowser:
		rightText = "[::b]Browser[::-]\n" + tview.Escape(strings.TrimSpace(consts.HelpPageBrowser))

	case PageQueue:
		rightText = "[::b]Queue[::-]\n" + tview.Escape(strings.TrimSpace(consts.HelpPageQueue))

	case PagePlaylists:
		rightText = "[::b]Playlists[::-]\n" + tview.Escape(strings.TrimSpace(consts.HelpPagePlaylists))

	case PageSearch:
		rightText = "[::b]Search[::-]\n" + tview.Escape(strings.TrimSpace(consts.HelpSearchPage))

	case PageLog:
		fallthrough
	default:
		// no text
	}

	h.rightColumn.SetText(rightText)

	h.helpBook.Clear()
	if rightText != "" {
		h.helpBook.AddItem(h.leftColumn, 38, 0, false).
			AddItem(h.rightColumn, 0, 1, true) // gets focus for scrolling
	} else {
		h.helpBook.AddItem(h.leftColumn, 0, 1, false)
	}
}
