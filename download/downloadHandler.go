package download

/*
1. downloadWAV(url String) ([] string,error)
+ getSongMetadata( url String) ([] track) ( SPOTIFY)
+ FindYoutubeUrl (tracks []track) ([]track) (YOUTUBE)
CONVERT track ---> videoURL []String
+ DownloadYoutubeAudio ( videoURL []string) ( [] string)
*/

func downloadWAV(url string) error {
	tracks, err := getTracksMetadata(url)
	if err != nil {
		return nil
	}

}
