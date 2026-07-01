package storage

import "database/sql"

type ReadWriteStore struct {
	db *sql.DB
}

func NewReadWriteStore(db *sql.DB) *ReadWriteStore {
	return &ReadWriteStore{db}
}

type ReadOnlyStore struct {
	db *sql.DB
}

func NewReadOnlyStore(db *sql.DB) *ReadOnlyStore {
	return &ReadOnlyStore{db}
}

type DBStore struct {
	primary *sql.DB
	replica *sql.DB
}

func NewDBStore(primary, replica *sql.DB) *DBStore {
	return &DBStore{primary, replica}
}
