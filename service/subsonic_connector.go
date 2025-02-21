package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/spezifisch/stmps/utils"
	webp "golang.org/x/image/webp"
)

var state struct {
	token string
}

func (c *SubsonicConnection) buildUrl(path string, params url.Values) string {
	query := url.Values{}
	if c.Conf().PlaintextAuth {
		query.Set("p", c.Conf().Password)
	} else {
		token, salt := authToken(c.Conf().Password)
		query.Set("t", token)
		query.Set("s", salt)
	}
	query.Set("u", c.Conf().Username)
	query.Set("v", c.Conf().ClientVersion)
	query.Set("c", c.Conf().ClientName)
	query.Set("f", "json")

	for k, v := range params {
		query[k] = v
	}

	return c.Conf().Host + path + "?" + query.Encode()
}

// requests
func (c *SubsonicConnection) GetServerInfo() (*SubsonicResponse, error) {
	url := c.buildUrl("/rest/ping", nil)
	return c.getResponse(url)
}

func (c *SubsonicConnection) GetIndexes() (*SubsonicResponse, error) {
	url := c.buildUrl("/rest/getIndexes", nil)
	return c.getResponse(url)
}

func (c *SubsonicConnection) GetArtist(id string) (*SubsonicResponse, error) {
	if cachedResponse, present := directoryCache[id]; present {
		return &cachedResponse, nil
	}

	params := url.Values{"id": []string{id}}
	url := c.buildUrl("/rest/getArtist", params)

	resp, err := c.getResponse(url)
	if err != nil {
		return resp, err
	}

	// on a sucessful request, cache the response
	if resp.Status == "ok" {
		directoryCache[id] = *resp
	}

	sort.Sort(resp.Directory.Entities)

	return resp, nil
}

func (c *SubsonicConnection) GetAlbum(id string) (*SubsonicResponse, error) {
	if cachedResponse, present := directoryCache[id]; present {
		// This is because Albums that were fetched as Directories aren't populated correctly
		if cachedResponse.Album.Name != "" {
			return &cachedResponse, nil
		}
	}

	params := url.Values{"id": []string{id}}
	url := c.buildUrl("/rest/getAlbum", params)
	resp, err := c.getResponse(url)
	if err != nil {
		return resp, err
	}

	// on a sucessful request, cache the response
	// TODO: this is crap, if we cache at this level it's means the rest of the app is eagerly fetching data from the service, it shouldn't as it has the most context to ask for fresh data. only pricy calls should be optimized here
	if resp.Status == "ok" {
		directoryCache[id] = *resp
	}

	sort.Sort(resp.Directory.Entities)

	return resp, nil
}

func (c *SubsonicConnection) GetMusicDirectory(id string) (*SubsonicResponse, error) {
	if cachedResponse, present := directoryCache[id]; present {
		return &cachedResponse, nil
	}

	params := url.Values{"id": []string{id}}
	url := c.buildUrl("/rest/getMusicDirectory", params)
	resp, err := c.getResponse(url)
	if err != nil {
		return resp, err
	}

	// on a sucessful request, cache the response
	if resp.Status == "ok" {
		directoryCache[id] = *resp
	}

	sort.Sort(resp.Directory.Entities)

	return resp, nil
}

// GetCoverArt fetches album art from the server, by ID. The results are cached,
// so it is safe to call this function repeatedly. If id is empty, an error
// is returned. If, for some reason, the server response can't be parsed into
// an image, an error is returned. This function can parse GIF, JPEG, and PNG
// images.
func (c *SubsonicConnection) GetCoverArt(id string) (image.Image, error) {
	if id == "" {
		return nil, fmt.Errorf("GetCoverArt: no ID provided")
	}
	if rv, ok := coverArts[id]; ok {
		return rv, nil
	}
	params := url.Values{"id": []string{id}, "f": []string{"image/png"}}
	url := c.buildUrl("/rest/getCoverArt", params)
	caller := "GetCoverArt"
	req, err := c.baseRequest("GetCoverArt", http.MethodGet, url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		coverArts[id] = nil
		return nil, fmt.Errorf("[%s] failed to make GET request: %v", caller, err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	} else {
		coverArts[id] = nil
		return nil, fmt.Errorf("[%s] response body is nil", caller)
	}

	if res.StatusCode != http.StatusOK {
		coverArts[id] = nil
		return nil, fmt.Errorf("[%s] unexpected status code: %d, status: %s", caller, res.StatusCode, res.Status)
	}

	if len(res.Header["Content-Type"]) == 0 {
		coverArts[id] = nil
		return nil, fmt.Errorf("[%s] unknown image type (no content-type from server)", caller)
	}
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		coverArts[id] = nil
		return nil, fmt.Errorf("[%s] failed to read response body: %v", caller, err)
	}
	var art image.Image
	switch res.Header["Content-Type"][0] {
	case "image/png":
		art, err = png.Decode(bytes.NewReader(responseBody))
	case "image/jpeg":
		art, err = jpeg.Decode(bytes.NewReader(responseBody))
	case "image/gif":
		art, err = gif.Decode(bytes.NewReader(responseBody))
	case "image/webp":
		art, err = webp.Decode(bytes.NewReader(responseBody))
	default:
		coverArts[id] = nil
		return nil, fmt.Errorf("[%s] unhandled image type %s: %v", caller, res.Header["Content-Type"][0], err)
	}
	if art != nil {
		// FIXME coverArts shouldn't grow indefinitely. Add some LRU cleanup after loading a few hundred cover arts.
		coverArts[id] = art
	}
	return art, err
}

func (c *SubsonicConnection) GetRandomSongs(Id string, randomType string) (*SubsonicResponse, error) {
	// TODO: move to the config validation, no need to check it over and over again

	// Set the default size for random/similar songs, clamped to 500
	size := "50"
	if c.Conf().RandomSongNumber > 0 && c.Conf().RandomSongNumber < 500 {
		size = strconv.FormatInt(int64(c.Conf().RandomSongNumber), 10)
	}

	switch randomType {
	case "similar":
		params := url.Values{"id": []string{Id}, "count": []string{size}}
		url := c.buildUrl("/rest/getSimilarSongs?", params)
		return c.getResponse(url)
	default: // "random" and everything else
		params := url.Values{"size": []string{size}}
		url := c.buildUrl("/rest/getRandomSongs", params)
		return c.getResponse(url)
	}
}

func (c *SubsonicConnection) ScrobbleSubmission(id string, isSubmission bool) (resp *SubsonicResponse, err error) {
	params := url.Values{"id": []string{id}, "submission": []string{strconv.FormatBool(isSubmission)}}
	url := c.buildUrl("/rest/scrobble", params)
	return c.getResponse(url)
}

func (c *SubsonicConnection) GetStarred() (*SubsonicResponse, error) {
	url := c.buildUrl("/rest/getStarred", nil)
	return c.getResponse(url)
}

func (c *SubsonicConnection) ToggleStar(id string, starredItems map[string]struct{}) (*SubsonicResponse, error) {
	params := url.Values{"id": []string{id}}
	var url string
	_, ok := starredItems[id]
	if ok {
		url = c.buildUrl("/rest/unstar", params)
	} else {
		url = c.buildUrl("/rest/star", params)
	}

	return c.getResponse(url)
}

func (c *SubsonicConnection) GetPlaylists() (*SubsonicResponse, error) {
	url := c.buildUrl("/rest/getPlaylists", nil)
	resp, err := c.getResponse(url)
	if err != nil {
		return resp, err
	}

	for i := 0; i < len(resp.Playlists.Playlists); i++ {
		playlist := &resp.Playlists.Playlists[i]

		if playlist.SongCount == 0 {
			continue
		}

		response, err := c.GetPlaylist(string(playlist.Id))
		if err != nil {
			return nil, err
		}

		playlist.Entries = response.Playlist.Entries
	}

	return resp, nil
}

func (c *SubsonicConnection) GetPlaylist(id string) (*SubsonicResponse, error) {
	params := url.Values{"id": []string{id}}
	url := c.buildUrl("/rest/getPlaylist", params)
	return c.getResponse(url)
}

// CreatePlaylist creates or updates a playlist on the server.
// If id is provided, the existing playlist with that ID is updated with the new song list.
// If name is provided, a new playlist is created with the song list.
// Either id or name _must_ be populated, or the function returns an error.
// If _both_ id and name are poplated, the function returns an error.
// songIds may be nil, in which case the new playlist is created empty, or all
// songs are removed from the existing playlist.
func (c *SubsonicConnection) CreatePlaylist(id, name string, songIds []string) (*SubsonicResponse, error) {
	if (id == "" && name == "") || (id != "" && name != "") {
		return nil, errors.New("CreatePlaylist: exactly one of id or name must be provided")
	}
	params := url.Values{}
	if id != "" {
		params.Set("id", id)
	} else {
		params.Set("name", name)
	}
	for _, sid := range songIds {
		params.Add("songId", sid)
	}
	url := c.buildUrl("/rest/createPlaylist", params)
	return c.getResponse(url)
}

func (c *SubsonicConnection) GetAuthToken(caller string) (string, string, error) {
	if c.Conf().Authentik && len(c.Conf().ClientId) > 0 {
		if len(token) == 0 {
			payload := fmt.Sprintf("grant_type=client_credentials&client_id=%s&username=%s&password=%s&scope=profile", c.Conf().ClientId, c.Conf().Username, c.Conf().Password)
			auth, err := http.NewRequest(http.MethodPost, c.Conf().AuthURL, strings.NewReader(payload))
			if err != nil {
				return "", "", fmt.Errorf("[%s] Could not create SSO auth request: %v", caller, err)
			}
			auth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			authRes, err := http.DefaultClient.Do(auth)
			if err != nil {
				return "", "", fmt.Errorf("[%s] Failed when generating SSO auth token: %v", caller, err)
			}
			if authRes.Body != nil {
				defer authRes.Body.Close()
			} else {
				return "", "", fmt.Errorf("[%s] SSO auth response body is nil", caller)
			}
			body, err := io.ReadAll(authRes.Body)
			if err != nil {
				return "", "", fmt.Errorf("[%s] failed to read SSO auth response body: %v", caller, err)
			}
			var authResponse AuthResponse
			err = json.Unmarshal(body, &authResponse)
			if err != nil {
				return "", "", fmt.Errorf("[%s] failed to unmarshal SSO auth response body: %v", caller, err)
			}
			token = authResponse.AccessToken
		}
		return "Authorization", "Bearer " + token, nil
	}
	return "", "", nil
}

func (c *SubsonicConnection) baseRequest(caller, method, requestUrl string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, requestUrl, body)
	if err != nil {
		return nil, fmt.Errorf("[%s] Could not create request: %v", caller, err)
	}
	header, value, err := c.GetAuthToken(caller)
	if err != nil {
		return nil, err
	}
	req.Header.Set(header, value)
	return req, nil
}

func (c *SubsonicConnection) getResponseBodyless(requestUrl string) error {
	caller := utils.FuncnameOnly(2)
	req, err := c.baseRequest(caller, http.MethodGet, requestUrl, nil)
	if err != nil {
		return fmt.Errorf("[%s] Could not create request: %v", caller, err)
	}
	_, err = http.DefaultClient.Do(req)
	return err
}

func (c *SubsonicConnection) getResponse(requestUrl string) (*SubsonicResponse, error) {
	caller := utils.FuncnameOnly(2)
	req, err := c.baseRequest(caller, http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("[%s] Could not create request: %v", caller, err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to make GET request: %v", caller, err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	} else {
		return nil, fmt.Errorf("[%s] response body is nil", caller)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[%s] unexpected status code: %d, status: %s", caller, res.StatusCode, res.Status)
	}

	responseBody, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("[%s] failed to read response body: %v", caller, readErr)
	}

	var decodedBody responseWrapper
	err = json.Unmarshal(responseBody, &decodedBody)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to unmarshal response body: %v", caller, err)
	}

	return &decodedBody.Response, nil
}

func (c *SubsonicConnection) DeletePlaylist(id string) error {
	params := url.Values{"id": []string{id}}
	url := c.buildUrl("/rest/deletePlaylist", params)
	return c.getResponseBodyless(url)
}

func (c *SubsonicConnection) AddSongToPlaylist(playlistId string, songId string) error {
	params := url.Values{"playlistId": []string{playlistId}, "songIdToAdd": []string{songId}}
	url := c.buildUrl("/rest/updatePlaylist", params)
	return c.getResponseBodyless(url)
}

func (c *SubsonicConnection) RemoveSongFromPlaylist(playlistId string, songIndex int) error {
	params := url.Values{"playlistId": []string{playlistId}, "songIndexToRemove": []string{strconv.Itoa(songIndex)}}
	url := c.buildUrl("/rest/updatePlaylist", params)
	return c.getResponseBodyless(url)
}

// note that this function does not make a request, it just formats the play url
// to pass to mpv
func (c *SubsonicConnection) GetPlayUrl(entity *SubsonicEntity) string {
	// TODO: is this needed

	// we don't want to call stream on a directory
	if entity.IsDirectory {
		return ""
	}

	params := url.Values{"id": []string{entity.Id}}
	return c.buildUrl("/rest/stream", params)
}

// Search uses the Subsonic search3 API to query a server for all songs that have
// ID3 tags that match the query. The query is global, in that it matches in any
// ID3 field.
// https://www.subsonic.org/pages/api.jsp#search3
func (c *SubsonicConnection) Search(searchTerm string, artistOffset, albumOffset, songOffset int) (*SubsonicResponse, error) {
	params := url.Values{
		"query":        []string{searchTerm},
		"artistOffset": []string{strconv.Itoa(artistOffset)},
		"albumOffset":  []string{strconv.Itoa(albumOffset)},
		"songOffset":   []string{strconv.Itoa(songOffset)},
	}
	url := c.buildUrl("/rest/search3", params)
	return c.getResponse(url)
}

// StartScan tells the Subsonic server to initiate a media library scan. Whether
// this is a deep or surface scan is dependent on the server implementation.
// https://subsonic.org/pages/api.jsp#startScan
func (c *SubsonicConnection) StartScan() error {
	url := c.buildUrl("/rest/startScan", nil)
	res, err := c.getResponse(url)
	if err != nil {
		return err
	} else if !res.ScanStatus.Scanning {
		return fmt.Errorf("server returned false for scan status on scan attempt")
	}
	return nil
}

func (c *SubsonicConnection) SavePlayQueue(queueIds []string, current string, position int) error {
	params := url.Values{"current": []string{current}, "position": []string{fmt.Sprintf("%d", position)}}
	for _, songId := range queueIds {
		params.Add("id", songId)
	}
	url := c.buildUrl("/rest/savePlayQueue", params)
	_, err := c.getResponse(url)
	return err
}

func (c *SubsonicConnection) LoadPlayQueue() (*SubsonicResponse, error) {
	url := c.buildUrl("/rest/getPlayQueue", nil)
	return c.getResponse(url)
}
