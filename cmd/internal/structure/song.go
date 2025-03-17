package structure

type Group struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique;not null"`
}

type Song struct {
	ID          int         `json:"id" gorm:"primaryKey"`
	Song        string      `json:"song" gorm:"not null"`
	GroupID     int         `json:"group_id"`
	Group       Group       `json:"group" gorm:"foreignKey:GroupID"`
	SongDetails SongDetails `json:"song_details" gorm:"foreignKey:SongID"`
}
