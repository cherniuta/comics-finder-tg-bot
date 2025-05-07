package db

import (
	"context"
	"log/slog"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"yadro.com/course/update/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {

	query := `INSERT INTO comics (id,url,words)
      VALUES ($1,$2,$3)`

	_, err := db.conn.Exec(query, comics.ID, comics.URL, pq.Array(comics.Words))

	return err
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {

	var (
		wordsNumber       int
		comicsNumber      int
		wordsUniqueNumber int
	)
	query := `SELECT COUNT(*) words FROM comics`

	err := db.conn.QueryRow(query).Scan(&wordsNumber)
	if err != nil {
		return core.DBStats{}, err
	}

	query = `SELECT COUNT(*) FROM comics`

	err = db.conn.QueryRow(query).Scan(&comicsNumber)
	if err != nil {
		return core.DBStats{}, err
	}

	query = `SELECT COUNT(DISTINCT word) 
         FROM comics, unnest(words) AS word`

	err = db.conn.QueryRow(query).Scan(&wordsUniqueNumber)
	if err != nil {
		return core.DBStats{}, err
	}

	return core.DBStats{
			WordsTotal:    wordsNumber,
			WordsUnique:   wordsUniqueNumber,
			ComicsFetched: comicsNumber},
		nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {

	query := `SELECT id FROM comics`

	indexIterator, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer indexIterator.Close()

	var indexes []int
	for indexIterator.Next() {
		var id int
		if err = indexIterator.Scan(&id); err != nil {
			return nil, err
		}
		indexes = append(indexes, id)
	}

	return indexes, nil
}

func (db *DB) Drop(ctx context.Context) error {

	var tables []string
	err := db.conn.Select(&tables, `SELECT table_name 
         FROM information_schema.tables 
         WHERE table_schema='public'`)
	if err != nil {
		return err
	}

	query := `TRUNCATE TABLE ` + strings.Join(tables, ",") + ` CASCADE`
	_, err = db.conn.Exec(query)
	if err != nil {
		return err
	}

	return nil
}
