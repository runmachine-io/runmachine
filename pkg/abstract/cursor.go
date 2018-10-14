package abstract

// Generic interface for anything that can iterate over rows. Purposefully made
// to function like database/sql package's Rows interface.  Useful for test
// mocking and abstracting a storage layer.
type Cursor interface {
	Next() bool
	Close() error
	Scan(dest ...interface{}) error
}
