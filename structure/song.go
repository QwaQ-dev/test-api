package structure

type Song struct {
	Id    int    `json:"id"`
	Song  string `json:"song"`
	Group string `json:"group"`
	SongDetails
}
