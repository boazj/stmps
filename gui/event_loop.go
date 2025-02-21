// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package gui

import (
	"time"

	"github.com/spezifisch/stmps/mpvplayer"
	"github.com/spezifisch/stmps/utils"
)

type eventLoop struct {
	// scrobbles are handled by background loop
	scrobbleNowPlaying      chan string
	scrobbleSubmissionTimer *time.Timer
}

func (ui *Ui) initEventLoops() {
	el := &eventLoop{
		scrobbleNowPlaying: make(chan string, 5),
	}
	ui.eventLoop = el

	// create reused timer to scrobble after delay
	el.scrobbleSubmissionTimer = time.NewTimer(0)
	if !el.scrobbleSubmissionTimer.Stop() {
		<-el.scrobbleSubmissionTimer.C
	}
}

func (ui *Ui) runEventLoops() {
	go ui.guiEventLoop()
	go ui.backgroundEventLoop()
}

// handle ui updates
func (ui *Ui) guiEventLoop() {
	ui.addStarredToList()
	events := 0.0
	fpsTimer := time.NewTimer(0)

	for {
		events++

		select {
		case <-fpsTimer.C:
			fpsTimer.Reset(10 * time.Second)
			// ui.logger.Printf("guiEventLoop: %f events per second", events/10.0)
			events = 0

		case msg := <-ui.logger.(*utils.LoggerImpl).Output: // TODO: probably should have something better here
			// handle log page output
			ui.logPage.Print(msg)

		case mpvEvent := <-ui.mpvEvents:
			events++

			// handle events from mpv wrapper
			switch mpvEvent.Type {
			case mpvplayer.EventStatus:
				if mpvEvent.Data == nil {
					continue
				}
				volume := ui.player.State.Volume
				position := ui.player.State.Position
				duration := ui.player.State.Duration
				ui.app.QueueUpdateDraw(func() {
					ui.topbar.SetPlayerState(volume, position, duration)
				})

			case mpvplayer.EventStopped:
				ui.logger.Info("mpvEvent: stopped")
				ui.app.QueueUpdateDraw(func() {
					ui.topbar.SetActivityStop()
					ui.queuePage.UpdateQueue()
				})

			case mpvplayer.EventPlaying, mpvplayer.EventUnpaused:
				// TODO: verify this means "starting to play" and not simply playing
				// this is relevant for starting to overall play but also song change
				ui.logger.Info("mpvEvent: playing")

				currentSong, err := ui.player.GetPlayingTrack()
				if err == nil {
					// currentSong = mpvEvent.Data.(mpvplayer.QueueItem) // TODO is this safe to access? maybe we need a copy
					// TODO: the data passed on the event should be the relevant details not the whole entity

					if mpvEvent.Type == mpvplayer.EventPlaying {
						// Update MprisPlayer with new track info
						if ui.mprisPlayer != nil {
							ui.mprisPlayer.OnSongChange(currentSong)
						}

						if ui.connection.Conf().Scrobble {
							// TODO: move outside of eventloop, scrobble shouldn't effect player performance and processing loop

							// scrobble "now playing" event (delegate to background event loop)
							ui.eventLoop.scrobbleNowPlaying <- currentSong.Id

							// scrobble "submission" after song has been playing a bit
							// see: https://www.last.fm/api/scrobbling
							// A track should only be scrobbled when the following conditions have been met:
							// The track must be longer than 30 seconds. And the track has been played for
							// at least half its duration, or for 4 minutes (whichever occurs earlier.)
							if currentSong.Duration > 30 {
								scrobbleDelay := currentSong.Duration / 2
								if scrobbleDelay > 240 {
									scrobbleDelay = 240
								}
								scrobbleDuration := time.Duration(scrobbleDelay) * time.Second

								ui.eventLoop.scrobbleSubmissionTimer.Reset(scrobbleDuration)
								ui.logger.Debug("scrobbler: timer started, %v", scrobbleDuration)
							} else {
								ui.logger.Debug("scrobbler: track too short")
							}
						}
					}
					ui.app.QueueUpdateDraw(func() {
						ui.topbar.SetActivityPlaying(currentSong.Artist, currentSong.Title)
						ui.queuePage.UpdateQueue()
					})
				}

			case mpvplayer.EventPaused:
				ui.logger.Info("mpvEvent: paused")

				currentSong, err := ui.player.GetPlayingTrack()
				if err == nil {
					// currentSong = mpvEvent.Data.(mpvplayer.QueueItem) // TODO is this safe to access? maybe we need a copy
					// TODO: the data passed on the event should be the relevant details not the whole entity

					ui.app.QueueUpdateDraw(func() {
						ui.topbar.SetActivityPause(currentSong.Artist, currentSong.Title)
					})
				}

			default:
				ui.logger.Warn("guiEventLoop: unhandled mpvEvent %v", mpvEvent)
			}
		}
	}
}

// loop for blocking background tasks that would otherwise block the ui
func (ui *Ui) backgroundEventLoop() {
	for {
		select {
		case songId := <-ui.eventLoop.scrobbleNowPlaying:
			// scrobble now playing
			if _, err := ui.connection.ScrobbleSubmission(songId, false); err != nil {
				ui.logger.Error("scrobble nowplaying", err)
			}

		case <-ui.eventLoop.scrobbleSubmissionTimer.C:
			// scrobble submission delay elapsed
			if currentSong, err := ui.player.GetPlayingTrack(); err != nil {
				// user paused/stopped
				ui.logger.Debug("not scrobbling: %v", err)
			} else {
				// it's still playing
				ui.logger.Debug("scrobbling: %s", currentSong.Id)
				if _, err := ui.connection.ScrobbleSubmission(currentSong.Id, true); err != nil {
					ui.logger.Error("scrobble submission", err)
				}
			}
		}
	}
}

func (ui *Ui) addStarredToList() {
	response, err := ui.connection.GetStarred()
	if err != nil {
		ui.logger.Error("addStarredToList", err)
	}

	for _, e := range response.Starred.Song {
		// We're storing empty struct as values as we only want the indexes
		// It's faster having direct index access instead of looping through array values
		ui.starIdList[e.Id] = struct{}{}
	}
	for _, e := range response.Starred.Album {
		ui.starIdList[e.Id] = struct{}{}
	}
	for _, e := range response.Starred.Artist {
		ui.starIdList[e.Id] = struct{}{}
	}
}
