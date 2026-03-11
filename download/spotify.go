package download

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// 1.- loadCredentials
// 2.- saveToken
// 3.- loadCachedToken
// 4.- accessToken
// 5.- request
// 6.- getID
// 7.- isValidPattern
// 8.- TrackInfo
// 9.- PlaylistInfo
// 10.- AlbumInfo
// 11.- resourceInfo
// 12.- jsonList
// 13.- buildTrack
// 14.- pagination
// 15.- proccessItems

// Main Function : + getSongMetadata( url String) ([] Track) ( SPOTIFY)
/*
	.GetSongMetadata(url String) ([] track)
		- AccessToken
		- SEND REQUEST (url String) ([] Track)



*/
const (
	tokenURL        = "https://accounts.spotify.com/api/token"
	cachedTokenPath = "token.json"
	trackPath       = "path.json"
	trackURLSpotify = "https://api.spotify.com/v1/tracks/%s"
	albumURLSpotify = "https://api.spotify.com/v1/albums/%s/tracks"
)

type Track struct {
	Title, Artist, Album string
	Artists              []string
	Duration             int
	YoutubeURL           string
}

type credentials struct {
	ClientID     string
	ClientSecret string
}
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type cachedToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("clientID or clientSecret is blank")
		return
	}
	cred := credentials{
		clientID,
		clientSecret,
	}
	log.Println(cred)

}

func getTracksMetadata(url string) ([]Track, error) {
	var (
		reSpotifyTrack = regexp.MustCompile(`open\.spotify\.com\/(?:intl-.+\/)?track\/([a-zA-Z0-9]{22})(\?si=[a-zA-Z0-9]{16})?`)
		reSpotifyAlbum = regexp.MustCompile(`open\.spotify\.com\/album\/([a-zA-Z0-9]{22})`)
	)
	matches := reSpotifyTrack.FindStringSubmatch(url)
	if matches != nil || len(matches) >= 2 {
		trackId := matches[1]
		endpoint := fmt.Sprintf(trackURLSpotify, trackId)
		body, err := request(endpoint)
		if err != nil {
			return nil, err
		}
		return getTrackInfoFromBody(body)
	}

	matches = reSpotifyAlbum.FindStringSubmatch(url)
	if matches != nil || len(matches) >= 2 {
		trackId := matches[1]
		endpoint := fmt.Sprintf(albumURLSpotify, trackId)
		body, err := request(endpoint)
		if err != nil {
			return nil, err
		}
		return getAlbumInfoFromBody(body) // hoặc AlbumInfo(body)
	}
	return nil, fmt.Errorf("URL does not match any Spotify URL")
}

// LoadCreds -->  Save Token -->

func LoadCredentials() (*credentials, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("clientID or clientSecret is blank")
		return nil, fmt.Errorf("CLIENT_ID or SECRET is blank")
	}
	return &credentials{
		clientID,
		clientSecret,
	}, nil
}

func AccessToken() (string, error) {
	// Try cached token
	token, err := LoadCachedToken()
	if err == nil && token != "" {
		return token, nil
	}

	// Request to API

	tokenResp, err := getTokenFromAPI()
	if err != nil {
		return "", err
	}
	if err := SaveTokenToFile(tokenResp.AccessToken, tokenResp.ExpiresIn); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func getTokenFromAPI() (tokenResponse, error) {
	cred, err := LoadCredentials()
	if err != nil {
		return tokenResponse{}, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	raw := cred.ClientID + ":" + cred.ClientSecret
	b64 := base64.StdEncoding.EncodeToString([]byte(raw))
	req.Header.Set("Authorization", "Basic "+b64)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return tokenResponse{}, fmt.Errorf("token request failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return tokenResponse{}, err
	}
	return tr, nil
}
func SaveTokenToFile(token string, expiresIn int) error {
	ct := cachedToken{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	}
	data, err := json.MarshalIndent(ct, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachedTokenPath, data, 0644)
}
func LoadCachedToken() (string, error) {
	data, err := os.ReadFile(cachedTokenPath)
	if err != nil {
		return "", err
	}
	var ct cachedToken
	if err := json.Unmarshal(data, &ct); err != nil {
		return "", err
	}
	if time.Now().After(ct.ExpiresAt) {
		return "", errors.New("token expired")
	}
	return ct.Token, nil
}

func request(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error on making the request")
	}

	bearer, err := AccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+bearer)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read body failed: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("HTTP error: %s\nBody: %s\n", resp.Status, string(body))
		return nil, fmt.Errorf("http error: %s", resp.Status)
	}
	return body, nil
}

func getTrackInfoFromBody(body []byte) ([]Track, error) {
	var result struct {
		Name     string `json:"name"`
		Duration int    `json:"duration_ms"`
		Album    struct {
			Name string `json:"name"`
		} `json:"album"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var allArtists []string
	for _, a := range result.Artists {
		allArtists = append(allArtists, a.Name)
	}
	mainArtist := ""
	if len(allArtists) > 0 {
		mainArtist = allArtists[0]
	}
	track := (&Track{
		Title:    result.Name,
		Artist:   mainArtist,
		Artists:  allArtists,
		Album:    result.Album.Name,
		Duration: result.Duration / 1000,
	}).buildTrack()
	//saveTracksJSON(trackPath, track)
	return []Track{*track}, nil
}

func getAlbumInfoFromBody(body []byte) ([]Track, error) {
	var result struct {
		Items []struct {
			Name     string `json:"name"`
			Duration int    `json:"duration_ms"`
			Artists  []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var tracks []Track
	for _, item := range result.Items {
		var artists []string
		for _, artist := range item.Artists {
			artists = append(artists, artist.Name)
		}

		mainArtist := ""
		if len(artists) > 0 {
			mainArtist = artists[0]
		}
		tracks = append(tracks, *(&Track{
			Title:    item.Name,
			Artist:   mainArtist,
			Artists:  artists,
			Duration: item.Duration / 1000,
			Album:    "",
		}).buildTrack())
	}

	return tracks, nil
}

func (t *Track) buildTrack() *Track {
	track := &Track{
		Title:    t.Title,
		Artist:   t.Artist,
		Artists:  t.Artists,
		Duration: t.Duration,
		Album:    t.Album,
	}

	return track
}

//func PlayListInfo(url string) ([]Track, error) {
//	re := regexp.MustCompile(`open\.download\.com\/(?:intl-[a-zA-Z-]+\/)?playlist\/([a-zA-Z0-9]{22})(?:\?si=[a-zA-Z0-9]+)?`)
//	matches := re.FindStringSubmatch(url)
//	if len(matches) != 2 {
//		return nil, errors.New("invalid playlist URL")
//	}
//	id := matches[1]
//	fmt.Println(id)
//	//var tracks []Track
//	//offset := 0
//	//limit := 100
//	endpoint := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", id)
//	statusCode, jsonResponse, err := request(endpoint)
//
//	if err != nil {
//		return nil, fmt.Errorf("request error: %w", err)
//	}
//	if statusCode != 200 {
//		return nil, fmt.Errorf("non-200 status: %d body=%s", statusCode, jsonResponse)
//	}
//	var pretty bytes.Buffer
//	if err := json.Indent(&pretty, []byte(jsonResponse), "", "  "); err != nil {
//		fmt.Println(jsonResponse) // fallback nếu JSON lỗi
//	} else {
//		fmt.Println(pretty.String())
//	}
//	return nil, nil
//}

func appendTracksJSONArray(path string, newTracks []Track) error {
	// đọc file nếu có
	var tracks []Track
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &tracks); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	tracks = append(tracks, newTracks...)

	// ghi lại (ghi đè)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(tracks)
}

//matches := re.FindStringSubmatch(url)
//if len(matches) != 2 {
//return nil, errors.New("invalid album URL")
//}
//id := matches[1]
//
//if len(matches) <= 2 {
//return nil, errors.New("invalid track URL")
//}
//id := matches[1]
