package structure

type SongDetails struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	SongID      uint   `gorm:"foreignKey" json:"song_id"`
	ReleaseDate string `json:"release_date"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}
