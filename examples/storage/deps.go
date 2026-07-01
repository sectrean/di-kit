package storage

import (
	"database/sql"

	"github.com/sectrean/di-kit"
)

type DBTag uint8

const (
	Primary DBTag = iota
	Replica
)

var Dependencies = di.Module{
	di.WithService(ConnectPrimaryDB,
		di.WithTag(Primary),
	),
	di.WithService(ConnectReplicaDB,
		di.WithTag(Replica),
	),
	di.WithService(NewReadWriteStore,
		di.WithTagged[*sql.DB](Primary),
	),
	di.WithService(NewReadOnlyStore,
		di.WithTagged[*sql.DB](Replica),
	),
	di.WithService(NewDBStore,
		di.WithTagged[*sql.DB](Primary),
		di.WithTagged[*sql.DB](Replica),
	),
}
