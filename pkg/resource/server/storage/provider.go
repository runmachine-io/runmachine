package storage

import (
	"database/sql"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/resource/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
)

type ProviderRecord struct {
	Provider *pb.Provider
	ID       int64
}

// TODO(jaypipes): Move this to a utility package/lib
// Returns a string containing the expression IN with one or more question
// marks for parameter interpolation. If numArgs argument is 3, the returned
// value would be "IN (?, ?, ?)"
func InParamString(numArgs int) string {
	resLen := 5 + ((numArgs * 3) - 2)
	res := make([]byte, resLen)
	res[0] = 'I'
	res[1] = 'N'
	res[2] = ' '
	res[3] = '('
	for x := 4; x < (resLen - 1); x++ {
		res[x] = '?'
		x++
		if x < (resLen - 1) {
			res[x] = ','
			x++
			res[x] = ' '
		}
	}
	res[resLen-1] = ')'
	return string(res)
}

// providerExists returns true if a provider with the UUID exists, false
// otherwise
func (s *Store) providerExists(
	uuid string,
) (bool, error) {
	_, err := s.ProviderGetByUuid(uuid)
	return err == nil, nil
}

// ProviderGetByUuid returns a provider record matching the supplied UUID. If
// no such record exists, returns ErrNotFound
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

// ProviderGetMatching returns provider records matching any of the supplied
// filters.
func (s *Store) ProvidersGetMatching(
	any []*pb.ProviderFilter,
) ([]*ProviderRecord, error) {
	// TODO(jaypipes): Validate that the slice of supplied ProviderFilters is
	// valid (for example, that the filter contains at least one UUID,
	// partition, or provider type filter...
	qargs := make([]interface{}, 0)
	qs := `SELECT
  p.id
, p.uuid AS provider_uuid
, part.uuid AS partition_uuid
, pt.code AS provider_type
, p.generation
FROM providers AS p
JOIN provider_types AS pt
 ON p.provider_type_id = pt.id
JOIN partitions AS part
 ON p.partition_id = part.id`
	if len(any) > 0 {
		qs += `
WHERE `
	}
	for x, filter := range any {
		if x > 0 {
			qs += `
OR
`
		}
		qs += "("
		exprAnd := false
		if filter.UuidFilter != nil {
			qs += "p.uuid " + InParamString(len(filter.UuidFilter.Uuids))
			for _, uuid := range filter.UuidFilter.Uuids {
				qargs = append(qargs, uuid)
			}
			exprAnd = true
		}
		if filter.PartitionFilter != nil {
			if exprAnd {
				qs += " AND "
			}
			qs += "part.uuid " + InParamString(len(filter.PartitionFilter.Uuids))
			for _, uuid := range filter.PartitionFilter.Uuids {
				qargs = append(qargs, uuid)
			}
			exprAnd = true
		}
		if filter.ProviderTypeFilter != nil {
			if exprAnd {
				qs += " AND "
			}
			qs += "pt.code " + InParamString(len(filter.ProviderTypeFilter.Codes))
			for _, code := range filter.ProviderTypeFilter.Codes {
				qargs = append(qargs, code)
			}
			exprAnd = true
		}
		qs += ")"
	}
	rows, err := s.DB().Query(qs, qargs...)
	if err != nil {
		s.log.ERR("failed to get providers: %s.\nSQL: %s", err, qs)
		return nil, err
	}
	recs := make([]*ProviderRecord, 0)
	for rows.Next() {
		rec := &ProviderRecord{
			Provider: &pb.Provider{},
		}
		if err := rows.Scan(
			&rec.ID,
			&rec.Provider.Uuid,
			&rec.Provider.Partition,
			&rec.Provider.ProviderType,
			&rec.Provider.Generation,
		); err != nil {
			panic(err.Error())
		}
		recs = append(recs, rec)
	}
	return recs, nil
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

	if !util.IsUuidLike(prov.Partition) {
		return nil, errors.ErrInvalidPartitionFormat
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
