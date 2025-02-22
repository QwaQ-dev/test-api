package structure

type SongDetails struct {
	Song_id      string `json:"song_id"`
	Release_date string `json:"release_date"`
	Text         string `json:"text"`
	Link         string `json:"link"`
}
