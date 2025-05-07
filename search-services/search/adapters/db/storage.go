package db

import (
	"context"
	"log/slog"

	"github.com/lib/pq"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/search/core"
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

func (db *DB) CheckDB() error {
	_, err := db.conn.Exec("SELECT 1 FROM comics LIMIT 1")
	return err
}

func (db *DB) Search(ctx context.Context, keyword string) ([]int, error) {
	var ids []int

	query := `SELECT id FROM comics WHERE $1 = ANY(words)`

	err := db.conn.Select(&ids, query, keyword)
	if err != nil {
		return ids, err
	}
	return ids, nil

}

type Comics struct {
	ID    int            `db:"id"`
	URL   string         `db:"url"`
	Words pq.StringArray `db:"words"`
}

func (db *DB) Get(ctx context.Context, id int) (core.Comics, error) {
	var comics Comics

	query := `SELECT url,words FROM comics WHERE id=$1`

	err := db.conn.Get(&comics, query, id)
	if err != nil {
		return core.Comics{}, err
	}

	return core.Comics{ID: id, URL: comics.URL, Words: comics.Words}, err

}

func (db *DB) MaxId(ctx context.Context) (int, error) {
	var id int
	query := `SELECT MAX(id) FROM comics`

	err := db.conn.Get(&id, query)

	return id, err

}
