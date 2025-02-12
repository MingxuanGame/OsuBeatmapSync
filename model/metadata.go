package model

import "fmt"

type BeatmapMetadata struct {
	Beatmap
	ApprovedDate   int64             `json:"approved_date"`
	SubmitDate     int64             `json:"submit_date"`
	LastUpdate     int64             `json:"last_update"`
	HasStoryboard  bool              `json:"has_storyboard"`
	HasVideo       bool              `json:"has_video"`
	CannotDownload bool              `json:"has_download"`
	NoAudio        bool              `json:"has_audio"`
	Link           map[string]string `json:"link"`
	Path           map[string]string `json:"path"`
}

type BeatmapsetMetadata struct {
	Beatmaps       map[int]BeatmapMetadata `json:"beatmaps"`
	BeatmapsetId   int                     `json:"beatmapset_id"`
	LastUpdate     int64                   `json:"last_update"`
	Link           map[string]string       `json:"link"`
	Path           map[string]string       `json:"path"`
	HasStoryboard  bool                    `json:"has_storyboard"`
	HasVideo       bool                    `json:"has_video"`
	CannotDownload bool                    `json:"has_download"`
	NoAudio        bool                    `json:"has_audio"`
}

type MetadataGameMode struct {
	UpdateTime int64
}

type Metadata struct {
	GameMode    map[GameMode]MetadataGameMode `json:"game_mode"`
	Beatmaps    map[int]BeatmapMetadata       `json:"beatmaps"`
	Beatmapsets map[int]BeatmapsetMetadata    `json:"beatmap_sets"`
}

func (b BeatmapsetMetadata) Equal(other BeatmapsetMetadata) bool {
	if b.LastUpdate < other.LastUpdate || len(b.Beatmaps) != len(other.Beatmaps) {
		return false
	}

	for k, sourceBeatmap := range b.Beatmaps {
		otherBeatmap := other.Beatmaps[k]
		if sourceBeatmap.Artist != otherBeatmap.Artist ||
			sourceBeatmap.ArtistUnicode != otherBeatmap.ArtistUnicode ||
			sourceBeatmap.Title != otherBeatmap.Title ||
			sourceBeatmap.TitleUnicode != otherBeatmap.TitleUnicode ||
			sourceBeatmap.GameMode != otherBeatmap.GameMode ||
			sourceBeatmap.LastUpdate != otherBeatmap.LastUpdate ||
			sourceBeatmap.Status != otherBeatmap.Status ||
			sourceBeatmap.HasStoryboard != otherBeatmap.HasStoryboard ||
			sourceBeatmap.HasVideo != otherBeatmap.HasVideo ||
			sourceBeatmap.CreatorId != otherBeatmap.CreatorId ||
			sourceBeatmap.DifficultyName != otherBeatmap.DifficultyName ||
			sourceBeatmap.StarRating != otherBeatmap.StarRating ||
			sourceBeatmap.CS != otherBeatmap.CS ||
			sourceBeatmap.AR != otherBeatmap.AR ||
			sourceBeatmap.OD != otherBeatmap.OD ||
			sourceBeatmap.HP != otherBeatmap.HP ||
			sourceBeatmap.BPM != otherBeatmap.BPM ||
			sourceBeatmap.MaxCombo != otherBeatmap.MaxCombo ||
			sourceBeatmap.HitLength != otherBeatmap.HitLength ||
			sourceBeatmap.TotalLength != otherBeatmap.TotalLength {
			return false
		}
	}
	return true
}

func (b BeatmapsetMetadata) String() string {
	var beatmap BeatmapMetadata
	for _, v := range b.Beatmaps {
		beatmap = v
		break
	}
	return fmt.Sprintf("%d %s - %s", b.BeatmapsetId, beatmap.Artist, beatmap.Title)
}
