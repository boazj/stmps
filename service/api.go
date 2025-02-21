// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package service

import (
	"encoding/json"
	"image"
	"strconv"
	"strings"

	"github.com/spezifisch/stmps/utils"
)

type SubsonicConnection struct {
	conf *utils.Config
}

var (
	directoryCache map[string]SubsonicResponse = make(map[string]SubsonicResponse)
	coverArts      map[string]image.Image      = make(map[string]image.Image)
	token          string                      = ""
)

func InitConnection(conf utils.ConfigProvider) *SubsonicConnection {
	return &SubsonicConnection{
		conf: conf.Conf(),
	}
}

func (s *SubsonicConnection) Conf() *utils.Config {
	return s.conf
}

func (s *SubsonicConnection) ClearCache() {
	directoryCache = make(map[string]SubsonicResponse)
}

func (s *SubsonicConnection) RemoveCacheEntry(key string) {
	delete(directoryCache, key)
}

type Ider interface {
	ID() string
}

// response structs
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IdToken     string `json:"id_token"`
}

type SubsonicError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type SubsonicArtist struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
}

func (s SubsonicArtist) ID() string {
	return s.Id
}

type SubsonicDirectory struct {
	Id       string           `json:"id"`
	Parent   string           `json:"parent"`
	Name     string           `json:"name"`
	Entities SubsonicEntities `json:"child"`
}

func (s SubsonicDirectory) ID() string {
	return s.Id
}

type SubsonicSongs struct {
	Song SubsonicEntities `json:"song"`
}

type SubsonicResults struct {
	Artist []Artist         `json:"artist"`
	Album  []Album          `json:"album"`
	Song   SubsonicEntities `json:"song"`
}

type ScanStatus struct {
	Scanning bool `json:"scanning"`
	Count    int  `json:"count"`
}

type PlayQueue struct {
	Current  string           `json:"current"`
	Position int              `json:"position"`
	Entries  SubsonicEntities `json:"entry"`
}

type Artist struct {
	Id         string  `json:"id"`
	Name       string  `json:"name"`
	AlbumCount int     `json:"albumCount"`
	Album      []Album `json:"album"`
}

func (s Artist) ID() string {
	return s.Id
}

type Album struct {
	Id            string           `json:"id"`
	Created       string           `json:"created"`
	ArtistId      string           `json:"artistId"`
	Artist        string           `json:"artist"`
	Artists       []Artist         `json:"artists"`
	DisplayArtist string           `json:"displayArtist"`
	Title         string           `json:"title"`
	Album         string           `json:"album"`
	Name          string           `json:"name"`
	SongCount     int              `json:"songCount"`
	Duration      int              `json:"duration"`
	PlayCount     int              `json:"playCount"`
	Genre         string           `json:"genre"`
	Genres        []Genre          `json:"genres"`
	Year          int              `json:"year"`
	Song          SubsonicEntities `json:"song"`
	CoverArt      string           `json:"coverArt"`
}

func (s Album) ID() string {
	return s.Id
}

type Genre struct {
	Name string `json:"name"`
}

type SubsonicEntity struct {
	Id          string   `json:"id"`
	IsDirectory bool     `json:"isDir"`
	Parent      string   `json:"parent"`
	Title       string   `json:"title"`
	ArtistId    string   `json:"artistId"`
	Artist      string   `json:"artist"`
	Artists     []Artist `json:"artists"`
	Duration    int      `json:"duration"`
	Track       int      `json:"track"`
	DiscNumber  int      `json:"discNumber"`
	Path        string   `json:"path"`
	CoverArtId  string   `json:"coverArt"`
}

func (s SubsonicEntity) ID() string {
	return s.Id
}

// Return the title if present, otherwise fallback to the file path
func (e SubsonicEntity) GetSongTitle() string {
	if e.Title != "" {
		return e.Title
	}

	// we get around the weird edge case where a path ends with a '/' by just
	// returning nothing in that instance, which shouldn't happen unless
	// subsonic is being weird
	if e.Path == "" || strings.HasSuffix(e.Path, "/") {
		return ""
	}

	lastSlash := strings.LastIndex(e.Path, "/")

	if lastSlash == -1 {
		return e.Path
	}

	return e.Path[lastSlash+1 : len(e.Path)]
}

// SubsonicEntities is a sortable list of entities.
// Directories are first, then in alphabelical order. Entities are sorted by
// track number, if they have track numbers; otherwise, they're sorted
// alphabetically.
type SubsonicEntities []SubsonicEntity

func (s SubsonicEntities) Len() int      { return len(s) }
func (s SubsonicEntities) Swap(i, j int) { s[j], s[i] = s[i], s[j] }
func (s SubsonicEntities) Less(i, j int) bool {
	// Directories are before tracks, alphabetically
	if s[i].IsDirectory {
		if s[j].IsDirectory {
			return s[i].Title < s[j].Title
		}
		return true
	}
	// Disk and track numbers are only relevant within the same parent
	if s[i].Parent == s[j].Parent {
		// sort first by DiskNumber
		if s[i].DiscNumber == s[j].DiscNumber {
			// Tracks on the same disk are sorted by track
			return s[i].Track < s[j].Track
		}
		return s[i].DiscNumber < s[j].DiscNumber
	}
	// If we get here, the songs are either from different albums, or else
	// they're on the same disk

	return s[i].Title < s[j].Title
}

type SubsonicIndex struct {
	Name    string           `json:"name"`
	Artists []SubsonicArtist `json:"artist"`
}

type SubsonicPlaylists struct {
	Playlists []SubsonicPlaylist `json:"playlist"`
}

type SubsonicPlaylist struct {
	Id        SubsonicId       `json:"id"`
	Name      string           `json:"name"`
	SongCount int              `json:"songCount"`
	Entries   SubsonicEntities `json:"entry"`
}

type SubsonicResponse struct {
	Status        string            `json:"status"`
	Version       string            `json:"version"`
	Indexes       []SubsonicIndex   `json:"indexes"`
	Directory     SubsonicDirectory `json:"directory"`
	RandomSongs   SubsonicSongs     `json:"randomSongs"`
	SimilarSongs  SubsonicSongs     `json:"similarSongs"`
	Starred       SubsonicResults   `json:"starred"`
	Playlists     SubsonicPlaylists `json:"playlists"`
	Playlist      SubsonicPlaylist  `json:"playlist"`
	Error         SubsonicError     `json:"error"`
	Artist        Artist            `json:"artist"`
	Album         Album             `json:"album"`
	SearchResults SubsonicResults   `json:"searchResult3"`
	ScanStatus    ScanStatus        `json:"scanStatus"`
	PlayQueue     PlayQueue         `json:"playQueue"`
}

type responseWrapper struct {
	Response SubsonicResponse `json:"subsonic-response"`
}

type SubsonicId string

func (si *SubsonicId) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		return json.Unmarshal(b, (*string)(si))
	}
	var i int
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}
	s := strconv.Itoa(i)
	*si = SubsonicId(s)
	return nil
}
