package model

type BeatmapStatus int

//goland:noinspection ALL
const (
	StatusGraveyard BeatmapStatus = iota - 2
	StatusWIP
	StatusPending
	StatusRanked
	StatusApproved
	StatusQualified
	StatusLoved
)

type GameMode int

//goland:noinspection ALL
const (
	GameModeOsu GameMode = iota
	GameModeTaiko
	GameModeCtb
	GameModeMania
)

type GenreId int

//goland:noinspection ALL
const (
	GenreAny GenreId = iota
	GenreUnspecified
	GenreVideoGame
	GenreAnime
	GenreRock
	GenrePop
	GenreOther
	GenreNovelty
	GenreHipHop GenreId = iota + 1
	GenreElectronic
	GenreMetal
	GenreClassical
	GenreFolk
	GenreJazz
)

type LanguageId int

//goland:noinspection ALL
const (
	LangAny LanguageId = iota
	LangUnspecified
	LangEnglish
	LangJapanese
	LangChinese
	LangInstrumental
	LangKorean
	LangFrench
	LangGerman
	LangSwedish
	LangSpanish
	LangItalian
	LangRussian
	LangPolish
	LangOther
)

type Beatmap struct {
	Status       BeatmapStatus `json:"approved,string"`
	SubmitDate   string        `json:"submit_date"`
	ApprovedDate string        `json:"approved_date"`
	LastUpdate   string        `json:"last_update"`

	Artist         string     `json:"artist"`
	BeatmapId      int        `json:"beatmap_id,string"`
	BeatmapsetId   int        `json:"beatmapset_id,string"`
	BPM            float32    `json:"bpm,string"`
	Creator        string     `json:"creator"`
	CreatorId      int        `json:"creator_id,string"`
	StarRating     float64    `json:"difficultyrating,string"`
	CS             float64    `json:"diff_size,string"` // key count in osu!mania
	OD             float64    `json:"diff_overall,string"`
	AR             float64    `json:"diff_approach,string"`
	HP             float64    `json:"diff_drain,string"`
	HitLength      int        `json:"hit_length,string"`
	Source         string     `json:"source"`
	GenreId        GenreId    `json:"genre_id,string"`
	LanguageId     LanguageId `json:"language_id,string"`
	Title          string     `json:"title"`
	TotalLength    int        `json:"total_length,string"`
	DifficultyName string     `json:"version"`
	FileMd5        string     `json:"file_md5"`
	GameMode       GameMode   `json:"mode,string"`
	Tags           string     `json:"tags"`
	CountNormal    int        `json:"count_normal,string"`
	CountSlider    int        `json:"count_slider,string"`
	CountSpinner   int        `json:"count_spinner,string"`
	MaxCombo       int        `json:"max_combo,string"`

	HasStoryboard  int `json:"storyboard,string"`
	HasVideo       int `json:"video,string"`
	CannotDownload int `json:"download_unavailable,string"`
	NoAudio        int `json:"audio_unavailable,string"`

	ArtistUnicode string `json:"artist_unicode"`
	TitleUnicode  string `json:"title_unicode"`
}
