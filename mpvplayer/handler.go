// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package mpvplayer

//#include <mpv/client.h>
import "C"

import (
	"unsafe"

	"github.com/supersonic-app/go-mpv"
)

type PropertyEvent struct {
	Name   string
	Format mpv.Format
	Data   unsafe.Pointer
}

func (p *Player) EventLoop() {
	if err := p.instance.ObserveProperty(0, string(PlaybackTime), mpv.FORMAT_INT64); err != nil {
		p.logger.Error("Observe1", err)
	}
	if err := p.instance.ObserveProperty(0, string(Duration), mpv.FORMAT_INT64); err != nil {
		p.logger.Error("Observe2", err)
	}
	if err := p.instance.ObserveProperty(0, string(Volume), mpv.FORMAT_INT64); err != nil {
		p.logger.Error("Observe3", err)
	}

	for evt := range p.mpvEvents {
		if evt == nil {
			// quit signal
			break
		} else if evt.Event_Id == mpv.EVENT_PROPERTY_CHANGE {
			if evt.Data == nil {
				p.logger.Debug("mpv.EventLoop (%s): Has nil Data", evt.Event_Id.String())
				continue
			}
			propChangeEvent := (*C.struct_mpv_event_property)(evt.Data)
			name := Property(C.GoString((*C.char)(propChangeEvent.name)))

			if mpv.Format(propChangeEvent.format) == mpv.FORMAT_NONE {
				continue
			}
			if name == PlaybackTime {
				position := p.getPlayerStateProperty(evt.Event_Id, PlaybackTime)
				p.State.Position = position
				p.remoteState.timePos = float64(position)
			} else if name == Duration {
				duration := p.getPlayerStateProperty(evt.Event_Id, Duration)
				p.State.Duration = duration
			} else if name == Volume {
				volume := p.getPlayerStateProperty(evt.Event_Id, Volume)
				p.State.Volume = volume
			}
			p.sendGuiDataEvent(EventStatus, StatusUpdate{})
		} else if evt.Event_Id == mpv.EVENT_END_FILE && !p.replaceInProgress {
			// we don't want to update anything if we're in the process of replacing the current track

			if p.stopped {
				// this is feedback for a user-requested stop
				// don't delete the first track so it gets started from the beginning when pressing play
				p.logger.Info("mpv.EventLoop: mpv stopped")
				p.stopped = true
				p.sendGuiEvent(EventStopped)
			} else {
				// advance queue and play next track
				if len(p.queue) > 0 {
					p.queue = p.queue[1:]
				}

				if len(p.queue) > 0 {
					if err := p.instance.Command([]string{"loadfile", p.queue[0].Uri}); err != nil {
						p.logger.Error("mpv.EventLoop: load next", err)
					}
				} else {
					// no remaining tracks
					p.logger.Info("mpv.EventLoop: stopping (auto)")
					p.stopped = true
					p.sendGuiEvent(EventStopped)
				}
			}
		} else if evt.Event_Id == mpv.EVENT_START_FILE {
			p.replaceInProgress = false
			p.stopped = false

			currentSong := QueueItem{}
			if len(p.queue) > 0 {
				currentSong = p.queue[0]
			}

			if paused, err := p.IsPaused(); err != nil {
				p.logger.Error("mpv.EventLoop: IsPaused", err)
			} else if !paused {
				p.sendGuiDataEvent(EventPlaying, currentSong)
			} else {
				p.sendGuiDataEvent(EventPaused, currentSong)
			}
		} else if evt.Event_Id == mpv.EVENT_IDLE || evt.Event_Id == mpv.EVENT_NONE {
			continue
		} else {
			p.logger.Warn("mpv.EventLoop: unhandled event id %v", evt.Event_Id)
			continue
		}
	}
}

func (p *Player) sendGuiEvent(typ UiEventType) {
	if p.eventConsumer != nil {
		p.eventConsumer.SendEvent(UiEvent{
			Type: typ,
			Data: nil,
		})
	}

	p.sendRemoteEvent(typ, nil)
}

func (p *Player) sendGuiDataEvent(typ UiEventType, data interface{}) {
	if p.eventConsumer != nil {
		p.eventConsumer.SendEvent(UiEvent{
			Type: typ,
			Data: data,
		})
	}

	p.sendRemoteEvent(typ, data)
}

func (p *Player) sendRemoteEvent(typ UiEventType, data interface{}) {
	switch typ {
	case EventStopped:
		defer func() {
			for _, cb := range p.cbOnStopped {
				cb()
			}
		}()

	case EventUnpaused:
		fallthrough
	case EventPlaying:
		defer func() {
			if data != nil {
				p.sendSongChange(data.(QueueItem))
			}
			for _, cb := range p.cbOnPlaying {
				cb()
			}
		}()

	case EventPaused:
		defer func() {
			if data != nil {
				p.sendSongChange(data.(QueueItem))
			}
			for _, cb := range p.cbOnPaused {
				cb()
			}
		}()

	case EventStatus:
		defer func() {
			for _, cb := range p.cbOnSeek {
				cb()
			}
		}()
	}
}

func (p *Player) sendSongChange(track QueueItem) {
	for _, cb := range p.cbOnSongChange {
		cb(&track)
	}
}
