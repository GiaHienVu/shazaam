package download

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

/*
Input: track
Func1: Tìm link URL từ Youtube ( metadata Track) (urlYoutbe string)
Func2: Tải track về dùng youtube dlp (url string) ( file mp3)
Output: file mp3
*/
func GetYoutubeURL(track Track) (int, string, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}
	key := os.Getenv("YOUTUBE_APIKEY")
	if key == "" {
		return 0, "", fmt.Errorf("Key is blank")
	}
	if track.Artists[0] == "" || track.Title == "" {
		return 0, "", fmt.Errorf("Song title or artist name is blank")
	}
	query := track.Artists[0] + " " + track.Title
	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=snippet&type=video&maxResults=1&key=%s&q=%s",
		key,
		url.QueryEscape(query),
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, "", fmt.Errorf("error on making the request")
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("error on getting response: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("copy body: %w", err)
	}

	return 0, "", fmt.Errorf("copy body: %w", err)
}

//func request(endpoint string) (int, string, error) {
//	req, err := http.NewRequest("GET", endpoint, nil)
//	if err != nil {
//		return 0, "", fmt.Errorf("error on making the request")
//	}
//
//	bearer, err := AccessToken()
//	if err != nil {
//		return 0, "", fmt.Errorf("failed to get access token: %w", err)
//	}
//	fmt.Println(bearer)
//	req.Header.Add("Authorization", "Bearer "+bearer)
//
//	resp, err := (&http.Client{}).Do(req)
//	if err != nil {
//		return 0, "", fmt.Errorf("error on getting response: %w", err)
//	}
//	defer resp.Body.Close()
//
//	body, err := io.ReadAll(resp.Body)
//	if err != nil {
//		return 0, "", fmt.Errorf("error on reading response: %w", err)
//	}
//	return resp.StatusCode, string(body), nil
//}
