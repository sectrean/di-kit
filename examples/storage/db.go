package storage

import (
	"context"
	"database/sql"
)

func ConnectPrimaryDB(_ context.Context) (*sql.DB, error) {
	// Connect to the primary database
	return sql.Open("primary-db", "dbname=mydb")
}

func ConnectReplicaDB(_ context.Context) (*sql.DB, error) {
	// Connect to the replica database
	return sql.Open("replica-db", "dbname=mydb")
}
