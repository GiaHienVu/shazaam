package main

import (
	"fmt"
	"shazam/test"
)

func main() {
	//fmt.Print(spotify.TrackInfo("https://open.spotify.com/track/14qNHvX8h6HoynFuV0RxWS?si=7583dc2959d741e6"))
	fmt.Print(test.PlayListInfo2("https://open.spotify.com/albums/3zVOHT1NsTDeG4NyjZFlbA?si=xzbysKj3R0WY5gwKCUiXPQ"))
}

// find(filePath) đã có filePATH MP3( GHI ÂM TỪ MIC ) -->Chuyển sang WAV --> Tạo finderprints --> Lưu vào DB
// download(URL SPOTIFY) -->
//
