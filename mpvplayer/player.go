// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package mpvplayer

import (
	"errors"
	"math/rand"
	"strconv"

	"github.com/spezifisch/stmps/logger"
	"github.com/spezifisch/stmps/remote"
	"github.com/supersonic-app/go-mpv"
)

type PlayerQueue []QueueItem

type Player struct {
	instance      *mpv.Mpv
	mpvEvents     chan *mpv.Event
	eventConsumer EventConsumer
	queue         PlayerQueue
	logger        logger.Logger

	replaceInProgress bool
	stopped           bool

	State struct {
		Volume   int64
		Position int64
		Duration int64
	}
	// player state
	remoteState struct {
		timePos float64
	}

	// callbacks
	cbOnPaused     []func()
	cbOnStopped    []func()
	cbOnPlaying    []func()
	cbOnSeek       []func()
	cbOnSongChange []func(remote.TrackInterface)
}

var _ remote.ControlledPlayer = (*Player)(nil)

func NewPlayer(logger logger.Logger, options map[string]string) (player *Player, err error) {
	m := mpv.Create()

	for opt, value := range options {
		if err = m.SetOptionString(opt, value); err != nil {
			return
		}
	}

	if err = m.Initialize(); err != nil {
		return
	}
	player = &Player{
		instance:          m,
		mpvEvents:         make(chan *mpv.Event),
		eventConsumer:     nil, // must be set by calling RegisterEventConsumer()
		queue:             make([]QueueItem, 0),
		logger:            logger,
		replaceInProgress: false,
		stopped:           true,
	}

	go player.mpvEngineEventHandler(m)
	return
}

func (p *Player) mpvEngineEventHandler(instance *mpv.Mpv) {
	for {
		evt := instance.WaitEvent(1)
		p.mpvEvents <- evt
	}
}

func (p *Player) Quit() {
	p.mpvEvents <- nil
	p.instance.TerminateDestroy()
}

func (p *Player) RegisterEventConsumer(consumer EventConsumer) {
	p.eventConsumer = consumer
}

func (p *Player) PlayNextTrack() error {
	if len(p.queue) >= 1 {
		// advance queue if any tracks left
		p.queue = p.queue[1:]

		if len(p.queue) > 0 {
			// replace currently playing song with next song
			if loaded, err := p.IsSongLoaded(); err != nil {
				p.logger.Error("PlayNextTrack", err)
			} else if loaded {
				p.replaceInProgress = true
				if err := p.temporaryStop(); err != nil {
					p.logger.Error("temporaryStop", err)
				}
				return p.instance.Command([]string{"loadfile", p.queue[0].Uri})
			}
		} else {
			// stop with empty queue
			if err := p.Stop(); err != nil {
				p.logger.Error("Stop", err)
			}
		}
	} else {
		// queue empty
		if err := p.Stop(); err != nil {
			p.logger.Error("Stop", err)
		}
	}
	return nil
}

func (p *Player) PlayUri(id, uri, title, artist, album string, duration, track, disc int, coverArtId string) error {
	p.queue = []QueueItem{{id, uri, title, artist, duration, album, track, coverArtId, disc}}
	p.replaceInProgress = true
	if ip, e := p.IsPaused(); ip && e == nil {
		if err := p.Pause(); err != nil {
			p.logger.Error("Pause", err)
		}
	}
	return p.instance.Command([]string{"loadfile", uri})
}

func (p *Player) Stop() error {
	p.logger.Info("stopping (user)")
	p.stopped = true
	return p.instance.Command([]string{"stop"})
}

func (p *Player) temporaryStop() error {
	return p.instance.Command([]string{"stop"})
}

func (p *Player) IsSongLoaded() (bool, error) {
	idle, err := p.getPropertyBool(IdleActive)
	return !idle, err
}

func (p *Player) IsPaused() (bool, error) {
	pause, err := p.getPropertyBool(Pause)
	return pause, err
}

func (p *Player) IsPlaying() (playing bool, err error) {
	if idle, err := p.getPropertyBool(IdleActive); err != nil {
	} else if paused, err := p.getPropertyBool(Pause); err != nil {
	} else {
		playing = !idle && !paused
	}
	return
}

func (p *Player) Test() {
	res, err := p.getPropertyBool(IdleActive)
	p.logger.Debug("res %v err %v", res, err)
}

// Pause toggles playing music
// If a song is playing, it is paused. If a song is paused, playing resumes.
// If stopped, the song starts playing.
// The state after the toggle is returned, or an error.
func (p *Player) Pause() (err error) {
	loaded, err := p.IsSongLoaded()
	if err != nil {
		return
	}
	paused, err := p.IsPaused()
	if err != nil {
		return
	}

	if loaded && !p.stopped {
		// toggle pause if not stopped
		err = p.instance.Command([]string{"cycle", "pause"})
		if err != nil {
			p.logger.Error("cycle pause", err)
			return
		}
		paused = !paused

		currentSong := QueueItem{}
		if len(p.queue) > 0 {
			currentSong = p.queue[0]
		}

		if paused {
			p.sendGuiDataEvent(EventPaused, currentSong)
		} else {
			p.sendGuiDataEvent(EventUnpaused, currentSong)
		}
	} else {
		if len(p.queue) > 0 {
			currentSong := p.queue[0]
			err = p.instance.Command([]string{"loadfile", currentSong.Uri})
			if err != nil {
				p.logger.Error("loadfile", err)
				return
			}

			if p.stopped {
				p.stopped = false
				if err = p.instance.SetProperty("pause", mpv.FORMAT_FLAG, false); err != nil {
					p.logger.Error("setprop pause", err)
				}

				// mpv will send start file event which also sends the gui event
				// p.sendGuiDataEvent(EventPlaying, currentSong)
			} else {
				p.sendGuiDataEvent(EventUnpaused, currentSong)
			}
		} else {
			p.stopped = true
			p.sendGuiEvent(EventStopped)
		}
	}

	return
}

func (p *Player) SetVolume(percentValue int) error {
	if percentValue > 100 {
		percentValue = 100
	} else if percentValue < 0 {
		percentValue = 0
	}

	return p.instance.SetProperty(string(Volume), mpv.FORMAT_INT64, percentValue)
}

func (p *Player) AdjustVolume(increment int) error {
	volume, err := p.getPropertyInt64(Volume)
	if err != nil {
		return err
	}

	return p.SetVolume(int(volume) + increment)
}

func (p *Player) Seek(increment int) error {
	return p.instance.Command([]string{"seek", strconv.Itoa(increment)})
}

// accessed from gui context
func (p *Player) ClearQueue() {
	if err := p.Stop(); err != nil {
		p.logger.Error("Stop", err)
	}
	p.queue = make([]QueueItem, 0) // TODO mutex queue access
}

func (p *Player) DeleteQueueItem(index int) {
	// TODO mutex queue access
	if index >= len(p.queue) {
		p.logger.Warn("DeleteQueueItem bad index %d (len %d)", index, len(p.queue))
	} else if len(p.queue) > 1 {
		if index == 0 {
			if err := p.PlayNextTrack(); err != nil {
				p.logger.Error("PlayNextTrack", err)
			}
		} else {
			p.queue = append(p.queue[:index], p.queue[index+1:]...)
		}
	} else {
		p.ClearQueue()
	}
}

func (p *Player) AddToQueue(item *QueueItem) {
	p.queue = append(p.queue, *item)
}

func (p *Player) MoveSongUp(index int) {
	if index < 1 {
		p.logger.Debug("MoveSongUp(%d) can't move top item", index)
		return
	}
	if index >= len(p.queue) {
		p.logger.Debug("MoveSongUp(%d) not that many songs in queue", index)
		return
	}
	p.queue[index-1], p.queue[index] = p.queue[index], p.queue[index-1]
}

func (p *Player) MoveSongDown(index int) {
	if index < 0 {
		p.logger.Debug("MoveSongUp(%d) invalid index", index)
		return
	}
	if index >= len(p.queue)-1 {
		p.logger.Debug("MoveSongUp(%d) can't move last song down", index)
		return
	}
	p.queue[index], p.queue[index+1] = p.queue[index+1], p.queue[index]
}

func (p *Player) Shuffle() {
	max := len(p.queue)
	for range max / 2 {
		ra := rand.Intn(max)
		rb := rand.Intn(max)
		p.queue[ra], p.queue[rb] = p.queue[rb], p.queue[ra]
	}
}

func (p *Player) GetQueueItem(index int) (QueueItem, error) {
	if index < 0 || index >= len(p.queue) {
		return QueueItem{}, errors.New("invalid queue entry")
	}
	return p.queue[index], nil
}

func (p *Player) GetQueueCopy() PlayerQueue {
	cpy := make(PlayerQueue, len(p.queue))
	copy(cpy, p.queue)
	return cpy
}

// accessed from background context
func (p *Player) GetPlayingTrack() (QueueItem, error) {
	paused, err := p.IsPaused()
	if err != nil {
		return QueueItem{}, err
	}
	if paused {
		return QueueItem{}, errors.New("not playing")
	}

	if len(p.queue) == 0 { // TODO mutex queue access
		return QueueItem{}, errors.New("queue empty")
	}
	currentSong := p.queue[0]
	return currentSong, nil
}

// remote.ControlledPlayer callbacks
func (p *Player) OnPaused(cb func()) {
	p.cbOnPaused = append(p.cbOnPaused, cb)
}

func (p *Player) OnStopped(cb func()) {
	p.cbOnStopped = append(p.cbOnStopped, cb)
}

func (p *Player) OnPlaying(cb func()) {
	p.cbOnPlaying = append(p.cbOnPlaying, cb)
}

func (p *Player) OnSeek(cb func()) {
	p.cbOnSeek = append(p.cbOnSeek, cb)
}

func (p *Player) OnSongChange(cb func(track remote.TrackInterface)) {
	p.cbOnSongChange = append(p.cbOnSongChange, cb)
}

func (p *Player) GetTimePos() float64 {
	return p.remoteState.timePos
}

func (p *Player) IsSeeking() (bool, error) {
	return false, nil
}

func (p *Player) SeekAbsolute(position int) error {
	return p.instance.Command([]string{"seek", strconv.Itoa(position), "absolute"})
}

func (p *Player) Play() error {
	if isPlaying, err := p.IsPlaying(); err != nil {
		return err
	} else if !isPlaying {
		return p.Pause()
	}
	return nil
}

func (p *Player) NextTrack() error {
	return p.PlayNextTrack()
}

func (p *Player) PreviousTrack() (err error) {
	if err = p.Stop(); err != nil {
		return
	}
	return p.Pause()
}
