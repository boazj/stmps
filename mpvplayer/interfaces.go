// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package mpvplayer

type UiEventType int

const (
	// song stopped at end of queue, data: nil
	EventStopped UiEventType = iota
	// new song started playing, data: QueueItem
	EventPlaying
	// unpaused/paused song, data: QueueItem
	EventUnpaused
	EventPaused
	// UI status update, data: StatusUpdate
	EventStatus
)

type UiEvent struct {
	Type UiEventType
	Data interface{}
}

type EventConsumer interface {
	// create event that goes from mpv backend (this package) to a UI frontend
	SendEvent(event UiEvent)
}
