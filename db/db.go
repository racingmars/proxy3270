package db

import (
	"github.com/boltdb/bolt"
)

type DB struct {
	db *bolt.DB
}

func New() (*DB, error) {
	db, err := bolt.Open("data.db", 0600, nil)
	return &DB{db: db}, err
}

func (db *DB) Close() error {
	return db.db.Close()
}
