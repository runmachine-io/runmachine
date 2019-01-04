package storage

import (
	"database/sql"
	"strings"

	"github.com/go-sql-driver/mysql"

	"github.com/runmachine-io/runmachine/pkg/logging"
	"github.com/runmachine-io/runmachine/pkg/resource/server/config"
)

type Store struct {
	log *logging.Logs
	cfg *config.Config
	db  *sql.DB
}

func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// DB returns a handle to a SQL database. The forceNew parameter indicates that
// a new DB handle will always be created even if the cached DB handle for the
// Store is not nil. The unsafe parameter indicates that the returned DB handle
// will have connections that accept multiple statements. If unsafe is true,
// the Store's cached DB handle is not returned.
func (s *Store) DB(forceNew bool, unsafe bool) (*sql.DB, error) {
	if !unsafe && !forceNew && s.db != nil {
		return s.db, nil
	}
	cfg, err := mysql.ParseDSN(
		strings.TrimPrefix(s.cfg.StorageDSN, "mysql://"),
	)
	if err != nil {
		return nil, err
	}
	if unsafe {
		cfg.MultiStatements = true
	}
	dsn := cfg.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// NOTE(jaypipes): sql.Open() doesn't actually connect to the DB or
		// anything, so any error here is likely an OOM error and so fatal...
		return nil, err
	}
	return db, nil
}

func New(log *logging.Logs, cfg *config.Config) (*Store, error) {
	s := &Store{
		log: log,
		cfg: cfg,
	}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}
