package repository

import (
	"github.com/jmoiron/sqlx"
)

type sqlxRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}
