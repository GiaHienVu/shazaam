package download

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

/*
Input: track
Func1: Tìm link URL từ Youtube ( metadata Track) (urlYoutbe string)
Func2: Tải track về dùng youtube dlp (url string) ( file mp3)
Output: file mp3
*/

//	type searchResult struct {
//		type items struct {
//			type id struct {
//				videoId string `json:"videoId"`
//
//			}
//		} `json:"items"`
//	}
const durationMatchThreshold = 5
const audioFormat = "wav"

type MetadataSongYoutube struct {
	Title      string
	Artist     string
	YoutubeURL string
	Duration   int
}

type SearchYoutubeResult struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

type ytVideosResp struct {
	Items []struct {
		ID             string `json:"id"`
		ContentDetails struct {
			Duration string `json:"duration"` // ISO 8601 e.g. "PT3M45S"
		} `json:"contentDetails"`
	} `json:"items"`
}

func GetYoutubeURL(track Track) error {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}
	key := os.Getenv("YOUTUBE_APIKEY")
	if key == "" {
		return fmt.Errorf("Key is blank")
	}
	if track.Artists[0] == "" || track.Title == "" {
		return fmt.Errorf("Song title or artist name is blank")
	}
	query := track.Artists[0] + " " + track.Title
	limit := 10
	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=id&type=video&videoCategoryId=10&maxResults=%d&key=%s&q=%s",
		limit,
		key,
		url.QueryEscape(query),
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("error on making the request")
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("error on getting response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("youtube search failed: status=%d body=%s", resp.StatusCode, "")
	}
	body, err := io.ReadAll(resp.Body) // <-- body lấy ở đây
	if err != nil {
		return err
	}

	videoIDs := ExtractVideoIDsFromSearchResponse(body)
	if err != nil {
		return err
	}

	_, videoIDsWithDuration, _ := FetchVideosContentDetails(key, videoIDs)
	//fmt.Println(videoIDsWithDuration)
	fmt.Println("Hello")
	songDuration := track.Duration
	allowedDurationRangeStart := songDuration - durationMatchThreshold // FIX: tính 1 lần
	allowedDurationRangeEnd := songDuration + durationMatchThreshold
	for _, item := range videoIDsWithDuration.Items {
		resultSongDuration, err := parseISO8601DurationSeconds(item.ContentDetails.Duration) // FIX: duration là ISO8601 (PT3M45S)
		if err != nil {
			continue
		}
		if resultSongDuration >= allowedDurationRangeStart && resultSongDuration <= allowedDurationRangeEnd {
			fmt.Println("INFO: ", fmt.Sprintf("Found song with id '%s'", item.ID))
			track.YoutubeURL = "https://www.youtube.com/watch?v=" + item.ID
			break
		}
	}
	url12, _ := DownloadAudio(track.YoutubeURL)
	fmt.Println(url12)
	return fmt.Errorf("No song founded: %w", err)

}

func ExtractVideoIDsFromSearchResponse(body []byte) []string {
	var r SearchYoutubeResult
	if err := json.Unmarshal(body, &r); err != nil {
		return nil
	}

	ids := make([]string, 0, len(r.Items))
	for _, it := range r.Items {
		if it.ID.VideoID != "" {
			ids = append(ids, it.ID.VideoID)
		}
	}
	return ids
}

func FetchVideosContentDetails(apiKey string, videoIDs []string) (int, ytVideosResp, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return 0, ytVideosResp{}, fmt.Errorf("api key is blank")
	}
	if len(videoIDs) == 0 {
		return 0, ytVideosResp{}, fmt.Errorf("videoIDs is empty")
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "www.googleapis.com",
		Path:   "/youtube/v3/videos",
	}
	q := url.Values{}
	q.Set("part", "contentDetails")
	q.Set("key", apiKey)
	q.Set("id", strings.Join(videoIDs, ","))
	u.RawQuery = q.Encode()
	endpoint := u.String()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, ytVideosResp{}, fmt.Errorf("make request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, ytVideosResp{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, ytVideosResp{}, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, ytVideosResp{}, fmt.Errorf(
			"youtube videos.list failed: status=%d body=%s",
			resp.StatusCode, string(body),
		)
	}

	var out ytVideosResp
	if err := json.Unmarshal(body, &out); err != nil {
		return resp.StatusCode, ytVideosResp{}, fmt.Errorf("unmarshal json: %w", err)
	}

	return resp.StatusCode, out, nil
}

func parseISO8601DurationSeconds(s string) (int, error) {
	re := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)
	m := re.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("invalid duration: %q", s)
	}

	toInt := func(x string) int {
		if x == "" {
			return 0
		}
		n, err := strconv.Atoi(x)
		if err != nil {
			return 0
		}
		return n
	}

	h := toInt(m[1])
	min := toInt(m[2])
	sec := toInt(m[3])
	return h*3600 + min*60 + sec, nil
}

func DownloadAudio(videoURL string) (string, error) {
	outDir := "out_audio"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	// base output path
	outPath := filepath.Join(outDir, "song."+audioFormat)

	// nếu tồn tại thì thêm _1, _2... để không đè
	if _, err := os.Stat(outPath); err == nil {
		ext := filepath.Ext(outPath)
		base := strings.TrimSuffix(outPath, ext)
		for i := 1; ; i++ {
			p := fmt.Sprintf("%s_%d%s", base, i, ext)
			if _, err := os.Stat(p); os.IsNotExist(err) {
				outPath = p
				break
			}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	cmd := exec.Command(
		"yt-dlp",
		"-f", "bestaudio",
		"--extract-audio",
		"--audio-format", audioFormat,
		"-o", outPath,
		videoURL,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}
	return outPath, nil
}
