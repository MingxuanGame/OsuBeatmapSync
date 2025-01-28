package model

type BeatmapMetadata struct {
	Status        BeatmapStatus     `json:"status"`
	Artist        string            `json:"artist"`
	ArtistUnicode string            `json:"artist_unicode"`
	Title         string            `json:"title"`
	TitleUnicode  string            `json:"title_unicode"`
	BeatmapsetId  int               `json:"beatmapset_id"`
	BeatmapId     int               `json:"beatmap_id"`
	GameMode      GameMode          `json:"gamemode"`
	Creator       string            `json:"creator"`
	Link          map[string]string `json:"link"`
	Path          map[string]string `json:"path"`
	LastUpdate    int64             `json:"last_update"`
	HasStoryboard bool              `json:"has_storyboard"`
	HasVideo      bool              `json:"has_video"`
}

type BeatmapsetMetadata struct {
	Beatmaps      map[int]BeatmapMetadata `json:"beatmaps"`
	BeatmapsetId  int                     `json:"beatmapset_id"`
	LastUpdate    int64                   `json:"last_update"`
	Link          map[string]string       `json:"link"`
	Path          map[string]string       `json:"path"`
	HasStoryboard bool                    `json:"has_storyboard"`
	HasVideo      bool                    `json:"has_video"`
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
	if b.LastUpdate < other.LastUpdate {
		return false
	}

	if len(b.Beatmaps) != len(other.Beatmaps) {
		return false
	}

	for k, sourceBeatmap := range b.Beatmaps {
		otherBeatmap := other.Beatmaps[k]
		if sourceBeatmap.Artist != otherBeatmap.Artist {
			return false
		}
		if sourceBeatmap.ArtistUnicode != otherBeatmap.ArtistUnicode {
			return false
		}
		if sourceBeatmap.Title != otherBeatmap.Title {
			return false
		}
		if sourceBeatmap.TitleUnicode != otherBeatmap.TitleUnicode {
			return false
		}
		if sourceBeatmap.GameMode != otherBeatmap.GameMode {
			return false
		}
		//if sourceBeatmap.Creator!=otherBeatmap.Creator{return false}
		if sourceBeatmap.LastUpdate != otherBeatmap.LastUpdate {
			return false
		}
		if sourceBeatmap.Status != otherBeatmap.Status {
			return false
		}
		//if len(sourceBeatmap.Link) != len(otherBeatmap.Link) {
		//	return false
		//}
		//for i, sourceLink := range sourceBeatmap.Link {
		//	if sourceLink != otherBeatmap.Link[i] {
		//		return false
		//	}
		//}
	}
	return true
}
