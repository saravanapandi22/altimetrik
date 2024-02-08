package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const (
	LastFmApiKey     = "4068d491ec1503b25fa393258ed6e715"
	MusixmatchApiKey = "9fce286a37383b22ffa010818b8222d6"
)

// LastFmResponse represents the response structure from Last.fm API
type LastFmResponse struct {
	TopTracks struct {
		Track []struct {
			Name   string `json:"name"`
			Artist struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"artist"`
		} `json:"track"`
	} `json:"tracks"`
}

// MusixMatchResponseHeader represents the response header structure from Musixmatch API
type MusixMatchResponseHeader struct {
	Message struct {
		Header struct {
			StatusCode int `json:"status_code"`
		} `json:"header"`
	} `json:"message"`
}

// MusixMatchResponse represents the response structure from Musixmatch API
type MusixMatchResponse struct {
	Message struct {
		Body struct {
			Lyrics struct {
				LyricsBody string `json:"lyrics_body"`
			} `json:"lyrics"`
		} `json:"body"`
	} `json:"message"`
}

// ArtistInfo represents the artist information
type ArtistInfo struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Image string `json:"image"`
}

// TrackInfo represents the complete track information
type TrackInfo struct {
	Track  string     `json:"track"`
	Artist ArtistInfo `json:"artist"`
	Lyrics string     `json:"lyrics"`
	Image  string     `json:"image"`
}

type TrackOverAllDetails struct {
	AllTrackInfo []TrackInfo
}

func main() {
	err := os.Setenv("LastFmApiKey", LastFmApiKey)
	if err != nil {
		return
	}

	err = os.Setenv("MusixmatchApiKey", MusixmatchApiKey)
	if err != nil {
		return
	}

	http.HandleFunc("/top-track", topTrackHandler)
	fmt.Println("Server listening on port 8080...")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}

func topTrackHandler(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	var TrackOverAllDetails TrackOverAllDetails
	if region == "" {
		http.Error(w, "Region is required", http.StatusBadRequest)
		return
	}

	// Fetch top track in the region from Last.fm API
	topTrack, err := fetchTopTrack(region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	TrackOverAllDetails.AllTrackInfo = make([]TrackInfo, 0)

	for _, regionalTracks := range topTrack.TopTracks.Track {

		// Fetch lyrics for the top track from Musixmatch API
		lyrics, err := fetchLyrics(regionalTracks.Name, regionalTracks.Artist.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Fetch artist information and image from Last.fm API
		artistInfo, err := fetchArtistInfo(regionalTracks.Artist.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if regionalTracks.Name != "" {
			// Combine all data into TrackInfo struct
			trackInfo := TrackInfo{
				Track:  regionalTracks.Name,
				Artist: artistInfo,
				Lyrics: lyrics,
				Image:  artistInfo.Image,
			}
			TrackOverAllDetails.AllTrackInfo = append(TrackOverAllDetails.AllTrackInfo, trackInfo)
		}
	}

	// Marshal response data to JSON
	response, err := json.Marshal(TrackOverAllDetails)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	// Set response headers and write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func fetchTopTrack(region string) (*LastFmResponse, error) {
	lastFmAPIKey := os.Getenv("LastFmApiKey")
	if lastFmAPIKey == "" {
		return nil, fmt.Errorf("LastFmApiKey environment variable not set")
	}

	lastFmURL := fmt.Sprintf("http://ws.audioscrobbler.com/2.0/?method=geo.gettoptracks&country=%s&api_key=%s&format=json", url.QueryEscape(region), lastFmAPIKey)
	resp, err := http.Get(lastFmURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch top track from Last.fm API: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from Last.fm Api: %v", err)
	}

	var lastFmResponse LastFmResponse
	if err := json.Unmarshal(body, &lastFmResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Last.fm response: %v", err)
	}

	if len(lastFmResponse.TopTracks.Track) == 0 {
		return nil, fmt.Errorf("no top tracks found for region: %s", region)
	}

	return &lastFmResponse, nil
}

func fetchLyrics(track, artist string) (string, error) {
	musixMatchAPIKey := os.Getenv("MusixmatchApiKey")
	if musixMatchAPIKey == "" {
		return "", fmt.Errorf("MusixmatchApiKey environment variable not set")
	}

	musixMatchURL := fmt.Sprintf("https://api.musixmatch.com/ws/1.1/matcher.lyrics.get?format=json&apikey=%s&q_track=%s&q_artist=%s", musixMatchAPIKey, url.QueryEscape(track), url.QueryEscape(artist))
	resp, err := http.Get(musixMatchURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch lyrics from Musixmatch API: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from Musixmatch API: %v", err)
	}

	var MusixMatchResponseHeader MusixMatchResponseHeader
	if err := json.Unmarshal(body, &MusixMatchResponseHeader); err != nil {
		return "", fmt.Errorf("failed to unmarshal Musixmatch response1: %v", err)
	}

	if MusixMatchResponseHeader.Message.Header.StatusCode != 200 {
		return "", nil
	}

	var musixMatchResponse MusixMatchResponse
	if err := json.Unmarshal(body, &musixMatchResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal Musixmatch response1: %v", err)
	}

	return musixMatchResponse.Message.Body.Lyrics.LyricsBody, nil
}

func fetchArtistInfo(artist string) (ArtistInfo, error) {
	lastFmAPIKey := os.Getenv("LastFmApiKey")
	if lastFmAPIKey == "" {
		return ArtistInfo{}, fmt.Errorf("LastFmApiKey environment variable not set")
	}

	lastFmURL := fmt.Sprintf("http://ws.audioscrobbler.com/2.0/?method=artist.getinfo&artist=%s&api_key=%s&format=json", url.QueryEscape(artist), lastFmAPIKey)

	resp, err := http.Get(lastFmURL)
	if err != nil {
		return ArtistInfo{}, fmt.Errorf("failed to fetch artist info from Last.fm API: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ArtistInfo{}, fmt.Errorf("failed to read response body: %v", err)
	}

	var artistInfo struct {
		Artist struct {
			Name  string `json:"name"`
			URL   string `json:"url"`
			Image []struct {
				Text string `json:"#text"`
			} `json:"image"`
		} `json:"artist"`
	}

	if err := json.Unmarshal(body, &artistInfo); err != nil {
		return ArtistInfo{}, fmt.Errorf("failed to unmarshal artist info response: %v", err)
	}

	if artistInfo.Artist.Name == "" {
		return ArtistInfo{}, fmt.Errorf("artist not found: %s", artist)
	}

	imageURL := ""
	if len(artistInfo.Artist.Image) > 0 {
		imageURL = artistInfo.Artist.Image[len(artistInfo.Artist.Image)-1].Text
	}

	return ArtistInfo{Name: artistInfo.Artist.Name, URL: artistInfo.Artist.URL, Image: imageURL}, nil
}
