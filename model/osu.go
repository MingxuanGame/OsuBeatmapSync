package model

type BeatmapStatus int

const (
	StatusGraveyard BeatmapStatus = -2
	StatusWIP       BeatmapStatus = -1
	StatusPending   BeatmapStatus = 0
	StatusRanked    BeatmapStatus = 1
	StatusApproved  BeatmapStatus = 2
	StatusQualified BeatmapStatus = 3
	StatusLoved     BeatmapStatus = 4
)

type GameMode int

const (
	GameModeOsu GameMode = iota
	GameModeTaiko
	GameModeCtb
	GameModeMania
)

type Beatmap struct {
	SubmitDate    string        `json:"submit_date"`
	ApprovedDate  string        `json:"approved_date"`
	LastUpdate    string        `json:"last_update"`
	Artist        string        `json:"artist"`
	ArtistUnicode string        `json:"artist_unicode"`
	Status        BeatmapStatus `json:"approved,string"`
	Mode          GameMode      `json:"mode,string"`
	BeatmapId     int           `json:"beatmap_id,string"`
	BeatmapsetId  int           `json:"beatmapset_id,string"`
	Title         string        `json:"title"`
	TitleUnicode  string        `json:"title_unicode"`
	Creator       string        `json:"creator"`

	HasStoryBoard int `json:"storyboard,string"`
	HasVideo      int `json:"video,string"`
	HasDownload   int `json:"download,string"`
	HasAudio      int `json:"audio,string"`
}
