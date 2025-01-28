package sql

import (
	"database/sql"
	"encoding/json"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
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

func writeBeatmap(tx *sql.Tx, metadata *BeatmapMetadata) error {
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO beatmaps (beatmap_id, status, artist, title, beatmapset_id, gamemode, creator, link, path, artist_unicode, title_unicode, last_update) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
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
	_, err = stmt.Exec(metadata.BeatmapId, metadata.Status, metadata.Artist, metadata.Title, metadata.BeatmapsetId, metadata.GameMode, metadata.Creator, string(link), string(path), metadata.ArtistUnicode, metadata.TitleUnicode, metadata.LastUpdate)
	if err != nil {
		return err
	}
	return nil
}

func writeBeatmapset(tx *sql.Tx, metadata *BeatmapsetMetadata) error {
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO beatmapsets (beatmapsetid, last_update, link, path) VALUES (?, ?, ?, ?)")
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
	_, err = stmt.Exec(metadata.BeatmapsetId, metadata.LastUpdate, string(link), string(path))
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
		beatmap_id INTEGER NOT NULL,
		status INTEGER NOT NULL,
		artist TEXT,
		title TEXT,
		beatmapset_id INTEGER NOT NULL,
		gamemode INTEGER NOT NULL,
		creator TEXT, link TEXT, "path" TEXT, artist_unicode TEXT, title_unicode text,
		last_update INTEGER DEFAULT 0,
		CONSTRAINT beatmaps_pk PRIMARY KEY (beatmap_id)
	)`)
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
		err := rows.Scan(&m.BeatmapId, &m.Status, &m.Artist, &m.Title, &m.BeatmapsetId, &m.GameMode, &m.Creator, &link, &path, &m.ArtistUnicode, &m.TitleUnicode, &m.LastUpdate)
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
		err := rows.Scan(&m.BeatmapsetId, &m.LastUpdate, &link, &path)
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
		beatmapsets[m.BeatmapsetId] = m
	}

	return Metadata{GameMode: mode, Beatmaps: beatmaps, Beatmapsets: beatmapsets}, nil
}
