package main

import (
	"database/sql"

	"github.com/mattn/go-sqlite3"
)

const initPragmas = `
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
PRAGMA cache_size = -20000;
PRAGMA foreign_keys = ON;
PRAGMA auto_vacuum = INCREMENTAL;
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 2147483648;
PRAGMA page_size = 8192;
`

func init() {
	sql.Register("sqlite3_pragma", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			_, err := conn.Exec(initPragmas, nil)
			return err
		},
	})
}
