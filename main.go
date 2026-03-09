package main

import (
	"fmt"
	"shazam/download"
)

func main() {
	track, _ := download.TrackInfo("https://open.spotify.com/track/4IO2X2YoXoUMv0M2rwomLC?si=1322a52c21bf46f4")
	//fmt.Print(download.AlbumInfo("https://open.spotify.com/album/2cWBwpqMsDJC1ZUwz813lo?si=EEEOqEXvTtqmwdsuBS72Nw"))
	fmt.Println(download.GetYoutubeURL(*track))
}

// find(filePath) đã có filePATH MP3( GHI ÂM TỪ MIC ) -->Chuyển sang WAV --> Tạo finderprints --> Lưu vào DB
// download(URL SPOTIFY) -->
//
