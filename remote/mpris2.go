// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package remote

import (
	"errors"
	"math"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/spezifisch/stmps/utils"
)

type MprisPlayer struct {
	dbus   *dbus.Conn
	player ControlledPlayer
	logger utils.Logger

	metadata map[string]interface{}
}

func RegisterMprisPlayer(player ControlledPlayer, logger_ utils.Logger) (mpp *MprisPlayer, err error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return
	}

	mpp = &MprisPlayer{
		dbus:   conn,
		player: player,
		logger: logger_,
		metadata: map[string]interface{}{
			"mpris:trackid":     "",
			"mpris:length":      int64(0),
			"xesam:album":       "",
			"xesam:albumArtist": "",
			"xesam:artist":      []string{},
			"xesam:composer":    []string{},
			"xesam:genre":       []string{},
			"xesam:title":       "",
			"xesam:trackNumber": int(0),
		},
	}

	mprisPlayer := map[string]*prop.Prop{
		"CanControl":     {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"CanGoNext":      {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"CanPause":       {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"CanPlay":        {Value: true, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"CanSeek":        {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"CanGoPrevious":  {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"Metadata":       {Value: mpp.metadata, Writable: false, Emit: prop.EmitTrue, Callback: nil},
		"Volume":         {Value: float64(0.0), Writable: true, Emit: prop.EmitTrue, Callback: mpp.volumeChange},
		"PlaybackStatus": {Value: "", Writable: false, Emit: prop.EmitFalse, Callback: nil},
	}

	mediaPlayer := map[string]*prop.Prop{
		"CanQuit":             {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"CanRaise":            {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"HasTrackList":        {Value: false, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"Identity":            {Value: "stmps", Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"IconName":            {Value: "stmps-icon", Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"SupportedUriSchemes": {Value: []string{}, Writable: false, Emit: prop.EmitFalse, Callback: nil},
		"SupportedMimeTypes":  {Value: []string{}, Writable: false, Emit: prop.EmitFalse, Callback: nil},
	}

	props, err := prop.Export(
		conn,
		"/org/mpris/MediaPlayer2",
		map[string]map[string]*prop.Prop{
			"org.mpris.MediaPlayer2":        mediaPlayer,
			"org.mpris.MediaPlayer2.Player": mprisPlayer,
		},
	)
	if err != nil {
		logger_.Error("prop.Export error", err)
		return
	}

	n := &introspect.Node{
		Name: "/org/mpris/MediaPlayer2",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: "org.mpris.MediaPlayer2.Player",
				Methods: []introspect.Method{
					{
						Name: "Next",
					},
					{
						Name: "Pause",
					},
					{
						Name: "Play",
					},
					{
						Name: "PlayPause",
					},
					{
						Name: "Stop",
					},
					{
						Name: "Seek",
						Args: []introspect.Arg{
							{Name: "Offset", Type: "x", Direction: "in"},
						},
					},
					{
						Name: "SetPosition",
						Args: []introspect.Arg{
							{Name: "TrackId", Type: "o", Direction: "in"},
							{Name: "Position", Type: "x", Direction: "in"},
						},
					},
				},
				Properties: props.Introspection("org.mpris.MediaPlayer2.Player"), // we implement the standard interface
			},
			{
				Name:       "org.mpris.MediaPlayer2",
				Methods:    []introspect.Method{},
				Properties: props.Introspection("org.mpris.MediaPlayer2"),
			},
		},
	}

	err = conn.Export(mpp, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	if err != nil {
		logger_.Error("conn.Export Player error", err)
		return
	}

	err = conn.Export(introspect.NewIntrospectable(n), "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Introspectable")
	if err != nil {
		logger_.Error("conn.Export Introspectable error", err)
		return
	}

	// our unique name
	name := "org.mpris.MediaPlayer2.stmps"
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		logger_.Error("conn.RequestName error", err)
		return
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		err = errors.New("name already owned")
		logger_.Error("conn.RequestName reply error", err)
		return
	}
	return
}

func (m *MprisPlayer) Close() {
	if err := m.dbus.Close(); err != nil {
		m.logger.Error("mpp Close", err)
	}
}

// Mandatory functions
func (m *MprisPlayer) Stop() {
	if err := m.player.Stop(); err != nil {
		m.logger.Error("mpp Stop", err)
	}
}

func (m *MprisPlayer) Next() *dbus.Error {
	if err := m.player.NextTrack(); err != nil {
		m.logger.Error("mpp Next", err)
		return dbus.MakeFailedError(err)
	}
	return nil
}

// set paused
func (m *MprisPlayer) Pause() *dbus.Error {
	if paused, err := m.player.IsPaused(); err != nil {
		m.logger.Error("mpp IsPaused", err)
		return dbus.MakeFailedError(err)
	} else if !paused {
		if err = m.player.Pause(); err != nil {
			m.logger.Error("mpp Pause", err)
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

// set playing
func (m *MprisPlayer) Play() *dbus.Error {
	if playing, err := m.player.IsPlaying(); err != nil {
		m.logger.Error("mpp IsPlaying", err)
		return dbus.MakeFailedError(err)
	} else if !playing {
		if err = m.player.Play(); err != nil {
			m.logger.Error("mpp Play", err)
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (m *MprisPlayer) PlayPause() *dbus.Error {
	if playing, err := m.player.IsPlaying(); err != nil {
		m.logger.Error("mpp IsPlaying", err)
		return dbus.MakeFailedError(err)
	} else if playing {
		if err = m.player.Pause(); err != nil {
			m.logger.Error("mpp Pause", err)
			return dbus.MakeFailedError(err)
		}
	} else {
		if err = m.player.Play(); err != nil {
			m.logger.Error("mpp Play", err)
			return dbus.MakeFailedError(err)
		}
	}
	return nil
}

func (m *MprisPlayer) Previous() *dbus.Error {
	// TODO not implemented
	return nil
}

func (m *MprisPlayer) Seek(offset int64, _ int) (int64, error) {
	// TODO not implemented
	return 0, nil
}

func (m *MprisPlayer) SetPosition(trackId dbus.ObjectPath, position int64) *dbus.Error {
	// TODO not implemented
	return nil
}

func (m *MprisPlayer) volumeChange(c *prop.Change) *dbus.Error {
	fVol := c.Value.(float64)

	// convert to %
	percentVol := int(math.Round(fVol * 100))
	if err := m.player.SetVolume(percentVol); err != nil {
		m.logger.Error("volumeChange", err)
		return dbus.MakeFailedError(err)
	} else {
		m.logger.Info("mpris: adjust volume %f -> %d%%", fVol, percentVol)
	}
	return nil
}

// OnSongChange method to be called by eventLoop
func (m *MprisPlayer) OnSongChange(currentSong TrackInterface) {
	m.metadata["mpris:trackid"] = "/org/mpris/MediaPlayer2/track/" + currentSong.GetId()
	m.metadata["mpris:length"] = int64(currentSong.GetDuration() * 1000000) // Duration in microseconds
	m.metadata["xesam:album"] = currentSong.GetAlbum()                      // Album name
	m.metadata["xesam:albumArtist"] = currentSong.GetAlbumArtist()          // Album artist
	m.metadata["xesam:artist"] = []string{currentSong.GetArtist()}          // List of artists
	m.metadata["xesam:composer"] = []string{}                               // List of composers, empty
	m.metadata["xesam:genre"] = []string{}                                  // List of genres, empty
	m.metadata["xesam:title"] = currentSong.GetTitle()                      // Track title
	m.metadata["xesam:trackNumber"] = currentSong.GetTrackNumber()          // Track number

	// m.logger.Printf("mpris: Updated metadata: %+v", m.metadata)

	// Emit the PropertiesChanged signal to notify clients about the metadata change
	err := m.dbus.Emit("/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Properties.PropertiesChanged",
		"org.mpris.MediaPlayer2.Player", map[string]interface{}{
			"Metadata": m.metadata,
		}, []string{})
	if err != nil {
		m.logger.Error("mpris: Emit PropertiesChanged", err)
	}
}
