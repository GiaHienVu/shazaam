package test

import (
	"bytes"
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
// CLIENT_ID="96026d7a2e3844a782626e9de78ee5d8"
// SECRET="d2497d5cda6a410db854b88bccb570c1"
// PORT="1212"
const (
	tokenURL        = "https://accounts.spotify.com/api/token"
	cachedTokenPath = "token.json"
	trackPath       = "path.json"
)

type Track struct {
	Title, Artist, Album string
	Artists              []string
	Duration             int
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

func request(endpoint string) (int, string, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, "", fmt.Errorf("error on making the request")
	}

	bearer, err := AccessToken()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get access token: %w", err)
	}
	fmt.Println(bearer)
	req.Header.Add("Authorization", "Bearer "+bearer)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("error on getting response: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("error on reading response: %w", err)
	}
	return resp.StatusCode, string(body), nil
}

func TrackInfo(url string) (*Track, error) {
	re := regexp.MustCompile(`open\.spotify\.com\/(?:intl-.+\/)?track\/([a-zA-Z0-9]{22})(\?si=[a-zA-Z0-9]{16})?`)
	matches := re.FindStringSubmatch(url)
	if len(matches) <= 2 {
		return nil, errors.New("invalid track URL")
	}
	id := matches[1]

	endpoint := fmt.Sprintf("https://api.spotify.com/v1/tracks/%s", id)
	statusCode, jsonResponse, err := request(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting track info: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", statusCode)
	}

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

	//var pretty bytes.Buffer
	//if err := json.Indent(&pretty, []byte(jsonResponse), "", "  "); err != nil {
	//	fmt.Println(jsonResponse) // fallback nếu JSON lỗi
	//} else {
	//	fmt.Println(pretty.String())
	//}
	if err := json.Unmarshal([]byte(jsonResponse), &result); err != nil {
		return nil, err
	}

	var allArtists []string
	for _, a := range result.Artists {
		allArtists = append(allArtists, a.Name)
	}
	track := (&Track{
		Title:    result.Name,
		Artist:   allArtists[0],
		Artists:  allArtists,
		Album:    result.Album.Name,
		Duration: result.Duration / 1000,
	}).buildTrack()
	//saveTracksJSON(trackPath, track)
	return track, nil
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

func PlayListInfo(url string) ([]Track, error) {
	re := regexp.MustCompile(`open\.spotify\.com\/(?:intl-[a-zA-Z-]+\/)?playlist\/([a-zA-Z0-9]{22})(?:\?si=[a-zA-Z0-9]+)?`)
	matches := re.FindStringSubmatch(url)
	if len(matches) != 2 {
		return nil, errors.New("invalid playlist URL")
	}
	id := matches[1]
	fmt.Println(id)
	//var tracks []Track
	//offset := 0
	//limit := 100
	endpoint := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", id)
	statusCode, jsonResponse, err := request(endpoint)

	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("non-200 status: %d body=%s", statusCode, jsonResponse)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, []byte(jsonResponse), "", "  "); err != nil {
		fmt.Println(jsonResponse) // fallback nếu JSON lỗi
	} else {
		fmt.Println(pretty.String())
	}
	return nil, nil
}

func PlayListInfo2(rawURL string) ([]Track, error) {
	u := strings.TrimSpace(rawURL)

	re := regexp.MustCompile(`(?:https?:\/\/)?open\.spotify\.com\/(?:intl-[a-zA-Z-]+\/)?albums\/([^?\/\s]+)`)
	matches := re.FindStringSubmatch(u)

	// debug để biết vì sao fail
	fmt.Printf("rawURL=%q\n", rawURL)
	fmt.Printf("trimmed=%q\n", u)
	fmt.Printf("matches=%#v\n", matches)

	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid album URL: %q", u)
	}

	id := matches[1]
	fmt.Println("album id:", id)

	endpoint := fmt.Sprintf("https://api.spotify.com/v1/albums/%s/tracks", id)
	statusCode, jsonResponse, err := request(endpoint)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("non-200 status: %d body=%s", statusCode, jsonResponse)
	}

	//var result struct {
	//	Name     string `json:"name"`
	//	Duration int    `json:"duration_ms"`
	//	Album    struct {
	//		Name string `json:"name"`
	//	} `json:"album"`
	//	Artists []struct {
	//		Name string `json:"name"`
	//	} `json:"artists"`
	//}
	//if err := json.Unmarshal([]byte(jsonResponse), &result); err != nil {
	//	return nil, err

	var result struct {
		Items []struct {
			Name     string `json:"name"`
			Duration int    `json:"duration_ms"`
			Artist   []struct {
				Name string `json:"name"`
			} `json:"artists"`
		}
	}

	if err := json.Unmarshal([]byte(jsonResponse), &result); err != nil {
		return nil, err
	}

	var tracks []Track
	for _, item := range result.Items {
		var artists []string
		for _, artist := range item.Artist {
			artists = append(artists, artist.Name)
		}
		tracks = append(tracks, *(&Track{
			Title:    item.Name,
			Artist:   artists[0],
			Artists:  artists,
			Duration: item.Duration / 1000,
			Album:    "",
		}).buildTrack())
	}
	saveTracksJSON(trackPath, tracks)

	return nil, nil
}

func saveTracksJSON(path string, tracks []Track) error {
	f, err := os.Create(path) // overwrite
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(tracks)
}
