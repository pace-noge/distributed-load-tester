package database

import (
	"github.com/pace-noge/distributed-load-tester/internal/domain"
)

// SharedLinkRepository implementation for PostgresDB is already in postgres.go
// This file is a placeholder for future custom logic if needed.

func NewSharedLinkRepository(db *PostgresDB) domain.SharedLinkRepository {
	return db
}
