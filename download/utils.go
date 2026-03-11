package download

import (
	"fmt"
	"io"
	"net/http"
)

func requestSpotify(endpoint string) ([]byte, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error on making the requestSpotify")
	}

	bearer, err := AccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+bearer)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("do requestSpotify failed: %w", err)
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

func requestYoutube(endpoint string) ([]byte, error) {

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error on making the requestSpotify")
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("error on getting response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube search failed: status=%d body=%s", resp.StatusCode, "")
	}
	body, err := io.ReadAll(resp.Body) // <-- body lấy ở đây
	if err != nil {
		return nil, err
	}
	return body, nil

}
