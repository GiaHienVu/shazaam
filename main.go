package main

import (
	"fmt"
	"shazam/download"
)

func main() {
	//fmt.Print(download.TrackInfo("https://open.spotify.com/track/14qNHvX8h6HoynFuV0RxWS?si=7583dc2959d741e6"))
	fmt.Print(download.AlbumInfo("https://open.spotify.com/album/2cWBwpqMsDJC1ZUwz813lo?si=EEEOqEXvTtqmwdsuBS72Nw"))
}

// find(filePath) đã có filePATH MP3( GHI ÂM TỪ MIC ) -->Chuyển sang WAV --> Tạo finderprints --> Lưu vào DB
// download(URL SPOTIFY) -->
//
