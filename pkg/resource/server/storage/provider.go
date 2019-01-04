package storage

import (
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/resource/proto"
)

type ProviderRecord struct {
	Provider *pb.Provider
	ID       int64
}

var (
	ErrUnknown   = fmt.Errorf("An unknown error occurred.")
	ErrDuplicate = fmt.Errorf("Record already exists.")
	ErrNotFound  = fmt.Errorf("No such record.")
)

// providerExists returns true if a provider with the UUID exists, false
// otherwise
func (s *Store) providerExists(
	uuid string,
) (bool, error) {
	_, err := s.ProviderGetByUuid(uuid)
	return err == nil, nil
}

// ProviderGetByUuid returns a provider record matching the supplied UUID. If no such record exists, returns ErrNotFound
func (s *Store) ProviderGetByUuid(
	uuid string,
) (*ProviderRecord, error) {
	qs := `SELECT
  p.id
, pt.code AS provider_type
, p.generation
FROM providers AS p
JOIN provider_types AS pt
 ON p.provider_type_id = pt.id
WHERE p.uuid = ?`
	qargs := []interface{}{uuid}
	rows, err := s.db.Query(qs, qargs...)
	if err != nil {
		return nil, err
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rec := &ProviderRecord{
		Provider: &pb.Provider{
			Uuid: uuid,
		},
	}
	for rows.Next() {
		err = rows.Scan(
			&rec.ID,
			&rec.Provider.ProviderType,
			&rec.Provider.Generation,
		)
		if err != nil {
			return nil, err
		}
	}
	return rec, nil
}

// ProviderCreate creates the provider record in backend storage and returns a
// ProviderRecord describing the new provider
func (s *Store) ProviderCreate(
	prov *pb.Provider,
) (*ProviderRecord, error) {
	exists, err := s.providerExists(prov.Uuid)
	if err != nil {
		s.log.ERR("failed looking up provider by UUID: %s", err)
		return nil, ErrUnknown
	}
	if exists {
		return nil, ErrDuplicate
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qargs := []interface{}{
		prov.Uuid,
		prov.ProviderType,
		1, // generation
	}

	qs := `
INSERT INTO providers (
  uuid
, provider_type_id
, generation
) VALUES (?, ?, ?)
`

	stmt, err := tx.Prepare(qs)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	res, err := stmt.Exec(qargs...)
	if err != nil {
		return nil, err
	}
	newId, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &ProviderRecord{
		Provider: prov,
		ID:       newId,
	}, nil
}
