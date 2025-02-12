package sql

import (
	"database/sql"
	"encoding/json"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"time"
)
import _ "github.com/ncruces/go-sqlite3/driver"
import _ "github.com/ncruces/go-sqlite3/embed"

const MetadataDBFilename = "metadata.db"

type Database struct {
	*sql.DB
}

func (d *Database) Close() error {
	return d.DB.Close()
}

func OpenDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return &Database{db}, nil
}

func writeGameModeUpdateTime(tx *sql.Tx, mode map[GameMode]MetadataGameMode) error {
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO update_time (gamemode, time) VALUES (?, ?)")
	if err != nil {
		return nil
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {

		}
	}(stmt)

	for gm, m := range mode {
		_, err := stmt.Exec(gm, m.UpdateTime)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeBeatmap(tx *sql.Tx, m *BeatmapMetadata) error {
	stmt, err := tx.Prepare(`INSERT INTO beatmaps (
			   BeatmapId, Status, SubmitDate,
		       ApprovedDate, LastUpdate, Artist,
		       BeatmapsetId, BPM, Creator,
		       CreatorId, StarRating,
		       CS, OD, AR, HP,
		       HitLength, Source, GenreId, LanguageId,
		       Title, TotalLength, DifficultyName, FileMd5,
		       GameMode, Tags, CountNormal, CountSlider,
		       CountSpinner, MaxCombo,
		       HasStoryboard, HasVideo, CannotDownload, NoAudio,
		       ArtistUnicode, TitleUnicode, Link, Path
		) VALUES (?, ?, ?,
		       	  ?, ?, ?,
		          ?, ?, ?,
                  ?, ?,
		          ?, ?, ?, ?,
		       	  ?, ?, ?, ?,
		          ?, ?, ?, ?,
                  ?, ?, ?, ?,
                  ?, ?,
                  ?, ?, ?, ?,
                  ?, ?, ?, ?)
`)
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {

		}
	}(stmt)

	link, err := json.Marshal(m.Link)
	if err != nil {
		return err
	}
	path, err := json.Marshal(m.Path)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(
		m.BeatmapId,
		m.Status,
		time.Unix(m.SubmitDate, 0).Format(time.DateTime),
		time.Unix(m.SubmitDate, 0).Format(time.DateTime),
		time.Unix(m.SubmitDate, 0).Format(time.DateTime),
		m.Artist,
		m.BeatmapsetId,
		m.BPM,
		m.Creator,
		m.CreatorId,
		m.StarRating,
		m.CS,
		m.OD,
		m.AR,
		m.HP,
		m.HitLength,
		m.Source,
		m.GenreId,
		m.LanguageId,
		m.Title,
		m.TotalLength,
		m.DifficultyName,
		m.FileMd5,
		m.GameMode,
		m.Tags,
		m.CountNormal,
		m.CountSlider,
		m.CountSpinner,
		m.MaxCombo,
		utils.Btoi(m.HasStoryboard),
		utils.Btoi(m.HasVideo),
		utils.Btoi(m.CannotDownload),
		utils.Btoi(m.NoAudio),
		m.ArtistUnicode,
		m.TitleUnicode,
		string(link),
		string(path),
	)
	if err != nil {
		return err
	}
	return nil
}

func writeBeatmapset(tx *sql.Tx, metadata *BeatmapsetMetadata) error {
	stmt, err := tx.Prepare("INSERT INTO beatmapsets (beatmapsetid, last_update, has_storyboard, has_video, cannot_download, no_audio, link, path) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {

		}
	}(stmt)

	link, err := json.Marshal(metadata.Link)
	if err != nil {
		return err
	}
	path, err := json.Marshal(metadata.Path)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(metadata.BeatmapsetId, metadata.LastUpdate, utils.Btoi(metadata.HasStoryboard), utils.Btoi(metadata.HasVideo), utils.Btoi(metadata.CannotDownload), utils.Btoi(metadata.NoAudio), string(link), string(path))
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) WriteMetadata(metadata *Metadata) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)

	err = writeGameModeUpdateTime(tx, metadata.GameMode)
	if err != nil {
		return err
	}
	for _, m := range metadata.Beatmaps {
		err = writeBeatmap(tx, &m)
		if err != nil {
			return err
		}
	}
	for _, m := range metadata.Beatmapsets {
		err = writeBeatmapset(tx, &m)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

//goland:noinspection SqlWithoutWhere
func (d *Database) DropAllMetadata() error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)

	// delete all data
	_, err = tx.Exec("DROP TABLE IF EXISTS beatmaps;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("DROP TABLE IF EXISTS update_time;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("DROP TABLE IF EXISTS beatmapsets;")
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE TABLE beatmaps (
	    BeatmapId INTEGER PRIMARY KEY,
	    Status INTEGER,
	    SubmitDate TEXT,
	    ApprovedDate TEXT,
	    LastUpdate TEXT,
	    Artist TEXT,
	    BeatmapsetId INTEGER,
	    BPM REAL,
	    Creator TEXT,
	    CreatorId INTEGER,
	    StarRating REAL,
	    CS REAL,
	    OD REAL,
	    AR REAL,
	    HP REAL,
	    HitLength INTEGER,
	    Source TEXT,
	    GenreId INTEGER,
	    LanguageId INTEGER,
	    Title TEXT,
	    TotalLength INTEGER,
	    DifficultyName TEXT,
	    FileMd5 TEXT,
	    GameMode INTEGER,
	    Tags TEXT,
	    CountNormal INTEGER,
	    CountSlider INTEGER,
	    CountSpinner INTEGER,
	    MaxCombo INTEGER,
	    HasStoryboard,
	    HasVideo INTEGER,
	    CannotDownload INTEGER,
	    NoAudio INTEGER,
	    ArtistUnicode TEXT,
	    TitleUnicode TEXT,
	    Link TEXT,
	    Path TEXT
	);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE "update_time" (
		gamemode INTEGER not null
			constraint update_time_pk
				primary key,
		time     INTEGER not null
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`create table beatmapsets(
		beatmapsetid INTEGER not null
			constraint beatmapsets_pk
				primary key,
		last_update  integer,
		has_storyboard integer,
		has_video    integer,
		cannot_download integer,
		no_audio    integer,
		link         text,
		path         text
	)`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d *Database) ReadMetadata() (Metadata, error) {
	rows, err := d.Query("SELECT * FROM update_time")
	if err != nil {
		return Metadata{}, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	mode := make(map[GameMode]MetadataGameMode)
	for rows.Next() {
		var gm GameMode
		var time int64
		err := rows.Scan(&gm, &time)
		if err != nil {
			return Metadata{}, err
		}
		mode[gm] = MetadataGameMode{UpdateTime: time}
	}

	rows, err = d.Query("SELECT * FROM beatmaps")
	if err != nil {
		return Metadata{}, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	beatmaps := make(map[int]BeatmapMetadata)
	for rows.Next() {
		var m BeatmapMetadata
		var link, path string
		var hasStoryboard, hasVideo, hasDownload, hasAudio int
		var _submitDate, _approvedDate, _lastUpdate string
		err := rows.Scan(&m.BeatmapId, &m.Status, &_submitDate, &_approvedDate, &_lastUpdate, &m.Artist, &m.BeatmapsetId, &m.BPM, &m.Creator, &m.CreatorId, &m.StarRating, &m.CS, &m.OD, &m.AR, &m.HP, &m.HitLength, &m.Source, &m.GenreId, &m.LanguageId, &m.Title, &m.TotalLength, &m.DifficultyName, &m.FileMd5, &m.GameMode, &m.Tags, &m.CountNormal, &m.CountSlider, &m.CountSpinner, &m.MaxCombo, &hasStoryboard, &hasVideo, &hasDownload, &hasAudio, &m.ArtistUnicode, &m.TitleUnicode, &link, &path)
		if err != nil {
			return Metadata{}, err
		}
		err = json.Unmarshal([]byte(link), &m.Link)
		if err != nil {
			return Metadata{}, err
		}
		err = json.Unmarshal([]byte(path), &m.Path)
		if err != nil {
			return Metadata{}, err
		}
		m.NoAudio = utils.Itob(hasAudio)
		m.CannotDownload = utils.Itob(hasDownload)
		m.HasStoryboard = utils.Itob(hasStoryboard)
		m.HasVideo = utils.Itob(hasVideo)
		submitDate, err := time.Parse(time.DateTime, _submitDate)
		if err != nil {
			return Metadata{}, err
		}
		m.SubmitDate = submitDate.Unix()
		approvedDate, err := time.Parse(time.DateTime, _approvedDate)
		if err != nil {
			return Metadata{}, err
		}
		m.ApprovedDate = approvedDate.Unix()
		lastUpdate, err := time.Parse(time.DateTime, _lastUpdate)
		if err != nil {
			return Metadata{}, err
		}
		m.LastUpdate = lastUpdate.Unix()
		beatmaps[m.BeatmapId] = m
	}

	rows, err = d.Query("SELECT * FROM beatmapsets")
	if err != nil {
		return Metadata{}, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	beatmapsets := make(map[int]BeatmapsetMetadata)
	for rows.Next() {
		var m BeatmapsetMetadata
		var link, path string
		var hasStoryboard, hasVideo, hasDownload, hasAudio int
		err := rows.Scan(&m.BeatmapsetId, &m.LastUpdate, &hasStoryboard, &hasVideo, &hasDownload, &hasAudio, &link, &path)
		if err != nil {
			return Metadata{}, err
		}
		err = json.Unmarshal([]byte(link), &m.Link)
		if err != nil {
			return Metadata{}, err
		}
		err = json.Unmarshal([]byte(path), &m.Path)
		if err != nil {
			return Metadata{}, err
		}
		m.NoAudio = utils.Itob(hasAudio)
		m.CannotDownload = utils.Itob(hasDownload)
		m.HasStoryboard = utils.Itob(hasStoryboard)
		m.HasVideo = utils.Itob(hasVideo)
		beatmapsets[m.BeatmapsetId] = m
	}

	return Metadata{GameMode: mode, Beatmaps: beatmaps, Beatmapsets: beatmapsets}, nil
}
