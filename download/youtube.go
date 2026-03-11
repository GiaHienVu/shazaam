package download

import (
	"encoding/json"
	"fmt"
	"log"
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

const (
	durationMatchThreshold = 5
	audioFormat            = "wav"
	limit                  = 10
)

func FindYoutubeUrl(tracks []Track) []Track {
	for i := range tracks {
		track := &tracks[i]
		if track.Artists[0] == "" || track.Title == "" {
			fmt.Println("Song title or artist name is blank")
			break
		}
		query := track.Artists[0] + " " + track.Title
		videoIds, err := GetYoutubeURL(query)
		if err != nil {
			fmt.Printf("Something goes wrong: %v\n", err)
			break
		}
		result, err := FetchVideosContentDetails(strings.Join(videoIds, ","))
		if err != nil {
			fmt.Printf("Something goes wrong: %v\n", err)
			break
		}
		finalURL, err := filterResultWithDuration(result, track.Duration)
		if err != nil {
			fmt.Printf("Something goes wrong: %v\n", err)
			break
		}
		track.YoutubeURL = finalURL
	}
	return tracks
}

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

func GetYoutubeURL(query string) ([]string, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}
	key := os.Getenv("YOUTUBE_APIKEY")
	if key == "" {
		return nil, fmt.Errorf("key is blank")
	}

	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=id&type=video&videoCategoryId=10&maxResults=%d&key=%s&q=%s",
		limit,
		key,
		url.QueryEscape(query),
	)

	body, err := requestYoutube(endpoint)
	if err != nil {
		return nil, err
	}
	return ExtractVideoIDsFromSearchResponse(body), nil
}

func ExtractVideoIDsFromSearchResponse(body []byte) []string {
	var result SearchYoutubeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	ids := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if item.ID.VideoID != "" {
			ids = append(ids, item.ID.VideoID)
		}
	}
	return ids
}

func FetchVideosContentDetails(query string) (ytVideosResp, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}
	key := os.Getenv("YOUTUBE_APIKEY")
	if key == "" {
		return ytVideosResp{}, fmt.Errorf("Key is blank")
	}

	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=contentDetails&key=%s&id=%s",
		key,
		url.QueryEscape(query),
	)

	body, err := requestYoutube(endpoint)
	if err != nil {
		return ytVideosResp{}, err
	}

	var result ytVideosResp
	if err := json.Unmarshal(body, &result); err != nil {
		return ytVideosResp{}, fmt.Errorf("unmarshal json: %w", err)
	}

	return result, nil
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

func filterResultWithDuration(result ytVideosResp, duration int) (string, error) {
	allowedDurationRangeStart := duration - durationMatchThreshold // FIX: tính 1 lần
	allowedDurationRangeEnd := duration + durationMatchThreshold
	for _, item := range result.Items {
		resultSongDuration, err := parseISO8601DurationSeconds(item.ContentDetails.Duration) // FIX: duration là ISO8601 (PT3M45S)
		if err != nil {
			continue
		}
		if resultSongDuration >= allowedDurationRangeStart && resultSongDuration <= allowedDurationRangeEnd {
			fmt.Println("INFO: ", fmt.Sprintf("Found song with id '%s'", item.ID))
			return "https://www.youtube.com/watch?v=" + item.ID, nil
		}
	}
	return "", fmt.Errorf("Cant find youtube URL")
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
