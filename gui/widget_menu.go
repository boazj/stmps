// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MenuWidget struct {
	Root *tview.Flex

	buttonsLeft  *tview.Flex
	buttonsRight *tview.Flex

	activeButton string
	buttons      map[string]*tview.Button

	buttonStyle     tcell.Style
	quitActiveStyle tcell.Style

	// external references
	ui *Ui
}

const (
	PAGE_BROWSER = iota
	PAGE_QUEUE
	PAGE_PLAYLISTS
	PAGE_SEARCH
	PAGE_LOG
)

var buttonOrder = []string{PageBrowser, PageQueue, PagePlaylists, PageSearch, PageLog}

func (ui *Ui) createMenuWidget() (m *MenuWidget) {
	m = &MenuWidget{
		activeButton: buttonOrder[PAGE_BROWSER],
		buttons:      make(map[string]*tview.Button),

		buttonStyle:     tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite),
		quitActiveStyle: tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorRed),

		ui: ui,
	}

	// page buttons on the left
	m.buttonsLeft = tview.NewFlex().
		SetDirection(tview.FlexColumn)
	m.createPageButtons()
	m.updatePageButtons()

	// help and quit button on the right
	quitButton := tview.NewButton("Q: quit").
		SetStyle(m.buttonStyle).
		SetActivatedStyle(m.quitActiveStyle).
		SetSelectedFunc(func() {
			ui.Quit()
		})

	helpButton := tview.NewButton("?: help").
		SetStyle(m.buttonStyle).
		SetActivatedStyle(m.buttonStyle).
		SetSelectedFunc(func() {
			ui.ShowHelp()
		})

	m.buttonsRight = tview.NewFlex().
		SetDirection(tview.FlexColumn)
	m.buttonsRight.AddItem(nil, 0, 1, false) // fill space to right-align the buttons
	m.buttonsRight.AddItem(helpButton, 9, 0, false)
	m.buttonsRight.AddItem(quitButton, 9, 0, false)

	m.Root = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(m.buttonsLeft, 0, 4, false).
		AddItem(m.buttonsRight, 0, 2, false)

	// clear background
	m.Root.Box = tview.NewBox()

	return
}

func (m *MenuWidget) createPageButtons() {
	for i, page := range buttonOrder {
		button := tview.NewButton(page)
		button.SetStyle(m.buttonStyle)
		// HACK because I couldn't find a way to un-focus a button after switching with 1,2,3,4 keys:
		button.SetActivatedStyle(m.buttonStyle)

		// create copy for our function
		buttonPage := page
		button.SetSelectedFunc(func() {
			m.ui.ShowPage(buttonPage)
		})

		m.buttons[page] = button
		// add button
		m.buttonsLeft.AddItem(button, 15, 0, false)

		// add spacer
		if i < len(buttonOrder)-1 {
			m.buttonsLeft.AddItem(nil, 1, 0, false)
		}
	}
}

func (m *MenuWidget) updatePageButtons() {
	for i, page := range buttonOrder {
		var text string
		if page == m.activeButton {
			text = fmt.Sprintf("%d: [::b]%s[::-]", i+1, page)
		} else {
			text = fmt.Sprintf("%d: %s", i+1, page)
		}

		m.buttons[page].SetLabel(text)
	}
}

func (m *MenuWidget) SetActivePage(name string) {
	if _, ok := m.buttons[name]; !ok {
		return // invalid button name
	}

	m.activeButton = name
	m.updatePageButtons()
}

func (m *MenuWidget) GetActivePage() string {
	return m.activeButton
}
