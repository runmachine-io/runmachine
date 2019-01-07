package storage

import (
	"database/sql"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/resource/proto"
)

type ProviderRecord struct {
	Provider *pb.Provider
	ID       int64
}

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
, part.uuid AS partition_uuid
, pt.code AS provider_type
, p.generation
FROM providers AS p
JOIN provider_types AS pt
 ON p.provider_type_id = pt.id
JOIN partitions AS part
 ON p.partition_id = part.id
WHERE p.uuid = ?`
	rec := &ProviderRecord{
		Provider: &pb.Provider{
			Uuid: uuid,
		},
	}
	err := s.DB().QueryRow(qs, uuid).Scan(
		&rec.ID,
		&rec.Provider.Partition,
		&rec.Provider.ProviderType,
		&rec.Provider.Generation,
	)
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.ErrNotFound
	case err != nil:
		log.Fatal(err)
	}
	return rec, nil
}

// ensurePartition creates a record in the partitions table for the supplied
// partition UUID if no such record exists and returns the newly-inserted
// partition record's internal identifier. If a partition record already exists
// for the UUID, the function just returns the internal identifier.
func (s *Store) ensurePartition(
	uuid string,
) (int64, error) {
	var id int64
	db := s.DB()
	qs := "SELECT id FROM partitions WHERE uuid = ?"
	err := db.QueryRow(qs, uuid).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		// New record. Create it and return the newly-created internal ID
		qs = "INSERT INTO partitions (uuid) VALUES (?)"
		res, err := db.Exec(qs, uuid)
		if err != nil {
			me, ok := err.(*mysql.MySQLError)
			if !ok {
				s.log.ERR("failed converting err to mysql.MYSQLError: %s", err)
				return 0, err
			}
			if me.Number == 1062 {
				// Another thread already inserted this partition, so just grab
				// the partition's internal ID
				qs := "SELECT id FROM partitions WHERE uuid = ?"
				err := db.QueryRow(qs, uuid).Scan(&id)
				if err != nil {
					s.log.ERR(
						"failed getting partition internal ID: %s",
						err,
					)
					return 0, err
				}
				return id, nil
			}
			s.log.ERR("failed getting partition internal ID: %s", me)
			return 0, err
		}
		s.log.L2("created new partitions record for UUID %s", uuid)
		return res.LastInsertId()
	case err != nil:
		return 0, err
	}
	return id, nil
}

// ensureProviderType creates a record in the provider_types table for the
// supplied provider_type code if no such record exists and returns the
// newly-inserted provider_type record's internal identifier. If a
// provider_type record already exists for the code, the function just returns
// the internal identifier.
func (s *Store) ensureProviderType(
	code string,
) (int64, error) {
	var id int64
	db := s.DB()
	qs := "SELECT id FROM provider_types WHERE code = ?"
	err := db.QueryRow(qs, code).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		// New record. Create it and return the newly-created internal ID
		qs = "INSERT INTO provider_types (code) VALUES (?)"
		res, err := db.Exec(qs, code)
		if err != nil {
			me, ok := err.(*mysql.MySQLError)
			if !ok {
				s.log.ERR("failed converting err to mysql.MYSQLError: %s", err)
				return 0, err
			}
			if me.Number == 1062 {
				// Another thread already inserted this provider_type, so just grab
				// the provider_type's internal ID
				qs := "SELECT id FROM provider_types WHERE code = ?"
				err := db.QueryRow(qs, code).Scan(&id)
				if err != nil {
					s.log.ERR(
						"failed getting provider_type internal ID: %s",
						err,
					)
					return 0, err
				}
				return id, nil
			}
			s.log.ERR("failed getting provider_type internal ID: %s", me)
			return 0, err
		}
		s.log.L2("created new provider_types record for code %s", code)
		return res.LastInsertId()
	case err != nil:
		return 0, err
	}
	return id, nil
}

// ProviderCreate creates the provider record in backend storage and returns a
// ProviderRecord describing the new provider
func (s *Store) ProviderCreate(
	prov *pb.Provider,
) (*ProviderRecord, error) {
	exists, err := s.providerExists(prov.Uuid)
	if err != nil {
		s.log.ERR("failed looking up provider by UUID: %s", err)
		return nil, errors.ErrUnknown
	}
	if exists {
		return nil, errors.ErrDuplicate
	}

	// Grab the internal IDs of the new provider's partition and provider type,
	// ensuring that records exist for the partition and provider type.
	partId, err := s.ensurePartition(prov.Partition)
	if err != nil {
		return nil, errors.ErrUnknown
	}
	ptId, err := s.ensureProviderType(prov.ProviderType)
	if err != nil {
		return nil, errors.ErrUnknown
	}

	tx, err := s.DB().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qs := `
INSERT INTO providers (
  uuid
, provider_type_id
, partition_id
, generation
) VALUES (?, ?, ?, ?)
`
	stmt, err := tx.Prepare(qs)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(
		prov.Uuid,
		ptId,
		partId,
		1, // generation
	)
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
