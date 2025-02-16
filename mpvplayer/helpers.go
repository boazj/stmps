// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package mpvplayer

import (
	"errors"

	"github.com/supersonic-app/go-mpv"
)

func (p *Player) getPlayerStateProperty(eid mpv.EventId, prop Property) int64 {
	value, err := p.getPropertyInt64(prop)
	if err != nil {
		p.logger.Printf("mpv.EventLoop (%s): GetProperty %s -- %s", eid, prop, err)
	}
	return value
}

func (p *Player) getPropertyInt64(name Property) (int64, error) {
	value, err := p.instance.GetProperty(string(name), mpv.FORMAT_INT64)
	if err != nil {
		return 0, err
	} else if value == nil {
		return 0, errors.New("nil value")
	}
	return value.(int64), err
}

func (p *Player) getPropertyBool(name Property) (bool, error) {
	value, err := p.instance.GetProperty(string(name), mpv.FORMAT_FLAG)
	if err != nil {
		return false, err
	} else if value == nil {
		return false, errors.New("nil value")
	}
	return value.(bool), err
}
