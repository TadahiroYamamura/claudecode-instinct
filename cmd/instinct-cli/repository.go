package main

import (
	"context"
	"database/sql"
)

type Repository interface {
	InsertInstinct(ctx context.Context, p InsertParams) (string, error)
}

type DoltRepository struct {
	conn *sql.Conn
}

func NewDoltRepository(conn *sql.Conn) *DoltRepository {
	return &DoltRepository{conn: conn}
}

func (r *DoltRepository) InsertInstinct(ctx context.Context, p InsertParams) (string, error) {
	return insertInstinct(ctx, r.conn, p)
}
