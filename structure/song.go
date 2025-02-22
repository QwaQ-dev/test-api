package structure

type Song struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
	SongDetails
}
