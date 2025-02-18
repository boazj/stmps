package gui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spezifisch/stmps/consts"
	"github.com/spezifisch/stmps/logger"
	"github.com/spezifisch/stmps/utils"
)

type TopBar struct {
	Row             *tview.Flex
	startStopStatus *tview.TextView
	playerStatus    *tview.TextView

	// external refs
	// ui     *Ui
	logger logger.Logger
}

func InitTopBar(logger logger.Logger) *TopBar {
	startStopStatus := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetScrollable(false)
	startStopStatus.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		return action, nil
	})

	playerStatus := tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetDynamicColors(true).
		SetScrollable(false)

	row := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(startStopStatus, 0, 1, false).
		AddItem(playerStatus, 20, 1, false)

	ret := &TopBar{
		Row:             row,
		startStopStatus: startStopStatus,
		playerStatus:    playerStatus,
		logger:          logger,
	}
	ret.setActivityBase()
	ret.SetPlayerState(0, 0, 0)

	return ret
}

func (t *TopBar) setActivityBase() {
	text := fmt.Sprintf("[::b]%s[::-] v%s", consts.ClientName, consts.ClientVersion)
	t.startStopStatus.SetText(text)
}

func (t *TopBar) SetActivityStop() {
	t.startStopStatus.SetText("[red::b]Stopped[::-]")
}

func (t *TopBar) SetActivityPlaying(artist string, title string) {
	if title == "" {
		title = "Unknown"
	}
	if artist == "" {
		artist = "Unknown"
	}
	title = tview.Escape(title)
	artist = tview.Escape(artist)
	text := fmt.Sprintf("[green::b]Playing[::-] [white]%s[::-] [gray]by[::-] [white]%s[::-]", title, artist)
	t.startStopStatus.SetText(text)
}

func (t *TopBar) SetActivityPause(artist string, title string) {
	if title == "" {
		title = "Unknown"
	}
	if artist == "" {
		artist = "Unknown"
	}
	title = tview.Escape(title)
	artist = tview.Escape(artist)
	text := fmt.Sprintf("[yellow::b]Paused[::-] [white]%s[::-] [gray]by[::-] [white]%s[::-]", title, artist)
	t.startStopStatus.SetText(text)
}

func (t *TopBar) SetPlayerState(volume int64, position int64, duration int64) {
	position = max(position, 0)
	duration = max(duration, 0)

	positionMin, positionSec := utils.SecondsToMinAndSec(position)
	durationMin, durationSec := utils.SecondsToMinAndSec(duration)

	text := fmt.Sprintf("[%d%%][::b][%02d:%02d/%02d:%02d]", volume, positionMin, positionSec, durationMin, durationSec)
	t.playerStatus.SetText(text)
}
