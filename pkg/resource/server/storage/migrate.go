package storage

import (
	"database/sql"
	"fmt"
	"log"
)

var (
	// A hash of DB driver -> list of forward-only schema and data migrations
	_MIGRATIONS = map[string][]string{
		"mysql": []string{
			`
CREATE TABLE partitions (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, uuid CHAR(32) NOT NULL
, UNIQUE INDEX uix_uuid (uuid)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE resource_types (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, code VARCHAR(200) NOT NULL
, description TEXT CHARACTER SET utf8 COLLATE utf8_bin NULL
, UNIQUE INDEX uix_code (code)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE capabilities (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, code VARCHAR(200) NOT NULL
, description TEXT CHARACTER SET utf8 COLLATE utf8_bin NULL
, UNIQUE INDEX uix_code (code)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE distance_types (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, code VARCHAR(200) NOT NULL
, description TEXT CHARACTER SET utf8 COLLATE utf8_bin NULL
, generation INT NOT NULL
, UNIQUE INDEX uix_code (code)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE distances (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, distance_type_id INT NOT NULL
, code VARCHAR(200) NOT NULL
, description TEXT CHARACTER SET utf8 COLLATE utf8_bin NULL
, position INT NOT NULL
, UNIQUE INDEX uix_distance_type_id (distance_type_id, code)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE provider_types (
  id SMALLINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, code VARCHAR(200) NOT NULL
, description TEXT CHARACTER SET utf8 COLLATE utf8_bin NULL
, UNIQUE INDEX uix_code (code)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE providers (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, uuid CHAR(32) NOT NULL
, provider_type_id INT NOT NULL
, generation INT UNSIGNED NOT NULL
, partition_id INT NOT NULL
, parent_provider_id BIGINT NULL
, UNIQUE INDEX uix_uuid (uuid)
, INDEX ix_partition_id_provider_type_id (partition_id, provider_type_id)
, INDEX ix_parent_provider_id (parent_provider_id)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE provider_trees (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, root_provider_id BIGINT NOT NULL
, nested_left INT NOT NULL
, nested_right INT NOT NULL
, generation INT NOT NULL
, UNIQUE INDEX uix_nested_sets (root_provider_id, nested_left, nested_right)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE provider_capabilities (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, provider_id INT NOT NULL
, capability_id INT NOT NULL
, UNIQUE INDEX uix_provider_capability (provider_id, capability_id)
, INDEX ix_capability (capability_id)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE inventories (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, provider_id BIGINT NOT NULL
, resource_type_id INT NOT NULL
, total BIGINT UNSIGNED NOT NULL
, reserved BIGINT UNSIGNED NOT NULL
, min_unit BIGINT UNSIGNED NOT NULL
, max_unit BIGINT UNSIGNED NOT NULL
, step_size BIGINT UNSIGNED NOT NULL
, allocation_ratio FLOAT NOT NULL
, UNIQUE INDEX uix_provider_id_resource_type_id (provider_id, resource_type_id)
, INDEX ix_resource_type_id_total (resource_type_id, total)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE provider_groups (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, uuid CHAR(32) NOT NULL
, UNIQUE INDEX uix_uuid (uuid)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE provider_group_members (
  provider_group_id INT NOT NULL
, provider_id BIGINT NOT NULL
, PRIMARY KEY (provider_group_id, provider_id)
, INDEX (provider_id)
);

CREATE TABLE provider_distances (
  id INT NOT NULL AUTO_INCREMENT PRIMARY KEY
, provider_id INT NOT NULL
, provider_group_id INT NOT NULL
, distance_id BIGINT NOT NULL
, UNIQUE INDEX uix_provider_provider_group_distance (
    provider_id
  , provider_group_id
  , distance_id)
);

CREATE TABLE consumer_types (
  id SMALLINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, code VARCHAR(200) NOT NULL
, description TEXT CHARACTER SET utf8 COLLATE utf8_bin NULL
, UNIQUE INDEX uix_code (code)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE consumers (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, consumer_type_id SMALLINT NOT NULL
, uuid CHAR(32) NOT NULL
, generation INT UNSIGNED NOT NULL
, owner_project_uuid CHAR(32) NOT NULL
, owner_user_uuid CHAR(32) NOT NULL
, UNIQUE INDEX uix_uuid (uuid)
, INDEX ix_consumer_type_id (consumer_type_id)
, INDEX ix_owner (owner_project_uuid, owner_user_uuid)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE allocations (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, consumer_id BIGINT NOT NULL
, acquire_time BIGINT NOT NULL
, release_time BIGINT NOT NULL
, INDEX ix_consumer_window (consumer_id, acquire_time, release_time)
, INDEX ix_window (acquire_time, release_time)
) CHARACTER SET latin1 COLLATE latin1_bin;

CREATE TABLE allocation_items (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY
, allocation_id BIGINT NOT NULL
, provider_id BIGINT NOT NULL
, resource_type_id INT NOT NULL
, used BIGINT UNSIGNED NOT NULL
, INDEX ix_allocation_id_provider_id_resource_type_id (
    allocation_id
  , provider_id
  , resource_type_id)
, INDEX ix_resource_type_id_provider_id_allocation_id (
    resource_type_id
  , provider_id
  , allocation_id)
) CHARACTER SET latin1 COLLATE latin1_bin;
`,
		},
	}
)

// migrate checks to see if the resource database schema is up to date, and if
// not, runs schema and data migrations to get the database up to the latest
// schema version
func (s *Store) migrate() error {
	// Grab a single-use DB handle that can support executing multiple
	// statements in an execution context
	db, err := s.DB(true, true)
	if err != nil {
		// The DSN is bad or we got an OOM, so it's appropriate to stop
		// execution here
		log.Fatalf("failed to get a DB handle during migration: %s\n", err)
	}
	defer db.Close()

	// TODO(jaypipes): Support retry with exponential backoff to allow for
	// out-of-order service startup
	if err := db.Ping(); err != nil {
		// The credentials for connecting to the DB are bad, so it's
		// appropriate to stop execution here
		log.Fatalf("failed to ping DB during migration: %s\n", err)
	}

	s.log.L3("determining current DB version...")

	if !tableExists(db, "db_version") {
		s.log.L3("resource DB is not versioned. initializing versioned DB...")
		if err := versionDB(db); err != nil {
			// If we couldn't create a simple DB table, it's appropriate to
			// stop execution here
			log.Fatalf(
				"failed to create db_version table during migration: %s\n",
				err,
			)
		}
		s.log.L1("initialized DB version")
	}

	return nil
}

// versionDB attempts to create the db_version table and sets the initial DB
// version to 0
func versionDB(db *sql.DB) error {
	qs := `
CREATE TABLE db_version (
	version INT UNSIGNED NOT NULL DEFAULT 0 PRIMARY KEY
);
INSERT INTO db_version (version) VALUES (0);
`
	if _, err := db.Query(qs); err != nil {
		return err
	}
	return nil
}

// tableExists returns true if the searched-for table exists in the DB, false
// otherwise. The supplied DB handle should have already been pinged to verify
// connectivity. This function swallows connectivity errors.
func tableExists(db *sql.DB, name string) bool {
	rows, err := db.Query("SHOW TABLES LIKE '" + name + "'")
	if err != nil {
		fmt.Printf("failed trying to check if table exists: %s\n", err)
		return false
	}
	for rows.Next() {
		return true
	}
	return false
}
