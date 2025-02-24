// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spezifisch/stmps/mpvplayer"
	"github.com/spezifisch/stmps/remote"
	"github.com/spezifisch/stmps/service"
	"github.com/spezifisch/stmps/utils"
)

// struct contains all the updatable elements of the Ui
type Ui struct {
	app   *tview.Application
	pages *tview.Pages

	// top bar
	// startStopStatus *tview.TextView
	// playerStatus    *tview.TextView

	topbar *TopBar

	// bottom bar
	menuWidget *MenuWidget

	// browser page
	browserPage *BrowserPage

	// queue page
	queuePage *QueuePage

	// playlist page
	playlistPage *PlaylistPage

	// search page
	searchPage *SearchPage

	// log page
	logPage *LogPage

	// modals
	addToPlaylistList    *tview.List
	messageBox           *tview.Modal
	helpModal            tview.Primitive
	helpWidget           *HelpWidget
	selectPlaylistModal  tview.Primitive
	selectPlaylistWidget *PlaylistSelectionWidget

	starIdList map[string]struct{}

	eventLoop   *eventLoop
	mpvEvents   chan mpvplayer.UiEvent
	mprisPlayer *remote.MprisPlayer

	playlists  []service.SubsonicPlaylist
	connection *service.SubsonicConnection
	player     *mpvplayer.Player
	logger     utils.Logger
}

const (
	// page identifiers (use these instead of hardcoding page names for showing/hiding)
	PageBrowser   = "Browser"
	PageQueue     = "Queue"
	PagePlaylists = "Playlists"
	PageSearch    = "Search"
	PageLog       = "Log"

	PageDeletePlaylist = "deletePlaylist"
	PageNewPlaylist    = "newPlaylist"
	PageAddToPlaylist  = "addToPlaylist"
	PageMessageBox     = "messageBox"
	PageHelpBox        = "helpBox"
	PageSelectPlaylist = "selectPlaylist"
)

func InitGui(indexes *[]service.SubsonicIndex,
	connection *service.SubsonicConnection,
	player *mpvplayer.Player,
	logger utils.Logger,
	mprisPlayer *remote.MprisPlayer,
) (ui *Ui) {
	ui = &Ui{
		starIdList: map[string]struct{}{},

		eventLoop: nil, // initialized by initEventLoops()
		mpvEvents: make(chan mpvplayer.UiEvent, 5),

		playlists:   []service.SubsonicPlaylist{},
		connection:  connection,
		player:      player,
		logger:      logger,
		mprisPlayer: mprisPlayer,
	}

	ui.initEventLoops()

	ui.app = tview.NewApplication()
	ui.pages = tview.NewPages()

	ui.menuWidget = ui.createMenuWidget()
	ui.helpWidget = ui.createHelpWidget()
	ui.selectPlaylistWidget = ui.createPlaylistSelectionWidget()

	// same as 'playlistList' except for the addToPlaylistModal
	// - we need a specific version of this because we need different keybinds
	ui.addToPlaylistList = tview.NewList().ShowSecondaryText(false)

	// message box for small notes
	ui.messageBox = tview.NewModal().
		SetText("hi there").
		SetBackgroundColor(tcell.ColorBlack)
	ui.messageBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		ui.pages.HidePage(PageMessageBox)
		return event
	})

	ui.selectPlaylistModal = makeModal(ui.selectPlaylistWidget.Root, 80, 5)

	// help box modal
	ui.helpModal = makeModal(ui.helpWidget.Root, 80, 30)
	ui.helpWidget.Root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Belts and suspenders. After the dialog is shown, this function will
		// _always_ be called. Therefore, check to ensure it's actually visible
		// before triggering on events. Also, don't close on every key, but only
		// ESC, like the help text says.
		if ui.helpWidget.visible && (event.Key() == tcell.KeyEscape) {
			ui.CloseHelp()
		}
		return event
	})

	ui.topbar = InitTopBar(logger)

	// browser page
	ui.browserPage = ui.createBrowserPage(indexes)

	// queue page
	ui.queuePage = ui.createQueuePage()

	// playlist page
	ui.playlistPage = ui.createPlaylistPage()

	// search page
	ui.searchPage = ui.createSearchPage()

	// log page
	ui.logPage = ui.createLogPage()

	ui.pages.AddPage(PageBrowser, ui.browserPage.Root, true, true).
		AddPage(PageQueue, ui.queuePage.Root, true, false).
		AddPage(PagePlaylists, ui.playlistPage.Root, true, false).
		AddPage(PageSearch, ui.searchPage.Root, true, false).
		AddPage(PageDeletePlaylist, ui.playlistPage.DeletePlaylistModal, true, false).
		AddPage(PageNewPlaylist, ui.playlistPage.NewPlaylistModal, true, false).
		AddPage(PageAddToPlaylist, ui.browserPage.AddToPlaylistModal, true, false).
		AddPage(PageSelectPlaylist, ui.selectPlaylistModal, true, false).
		AddPage(PageMessageBox, ui.messageBox, true, false).
		AddPage(PageHelpBox, ui.helpModal, true, false).
		AddPage(PageLog, ui.logPage.Root, true, false)

	rootFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.topbar.Row, 1, 0, false).
		AddItem(ui.pages, 0, 1, true).
		AddItem(ui.menuWidget.Root, 1, 0, false)

	// add main input handler
	rootFlex.SetInputCapture(ui.handlePageInput)

	ui.app.SetRoot(rootFlex, true).
		SetFocus(rootFlex).
		EnableMouse(true)

	ui.playlistPage.UpdatePlaylists()

	return ui
}

func (ui *Ui) Run() error {
	// receive events from mpv wrapper
	ui.player.RegisterEventConsumer(ui)

	// run gui/background event handler
	ui.runEventLoops()

	// run mpv event handler
	go ui.player.EventLoop()

	// gui main loop (blocking)
	return ui.app.Run()
}

func (ui *Ui) ShowHelp() {
	activePage := ui.menuWidget.GetActivePage()
	ui.helpWidget.RenderHelp(activePage)

	ui.pages.ShowPage(PageHelpBox)
	ui.pages.SendToFront(PageHelpBox)
	ui.app.SetFocus(ui.helpModal)
	ui.helpWidget.visible = true
}

func (ui *Ui) CloseHelp() {
	ui.helpWidget.visible = false
	ui.pages.HidePage(PageHelpBox)
}

func (ui *Ui) ShowSelectPlaylist() {
	ui.pages.ShowPage(PageSelectPlaylist)
	ui.pages.SendToFront(PageSelectPlaylist)
	ui.app.SetFocus(ui.selectPlaylistModal)
	ui.selectPlaylistWidget.visible = true
}

func (ui *Ui) CloseSelectPlaylist() {
	ui.pages.HidePage(PageSelectPlaylist)
	ui.selectPlaylistWidget.visible = false
}

func (ui *Ui) showMessageBox(text string) {
	ui.pages.ShowPage(PageMessageBox)
	ui.messageBox.SetText(text)
	ui.app.SetFocus(ui.messageBox)
}
