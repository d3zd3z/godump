// Schema for an SQL database.

package pool

import (
	"database/sql"
	"errors"
)

// A desired database schema.
type schema struct {
	version string
	schema  []string
	compats []schemaCompat
}

// For compatibility with older schemas, each can have an associated
// set of strings indicating features not present that the client can
// check.
type schemaCompat struct {
	version     string
	inabilities []string
}

// Attempt to set the schema for this database.
func setSchema(db *sql.DB, schema *schema) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	for _, line := range schema.schema {
		_, err = db.Exec(line)
		if err != nil {
			_ = tx.Rollback()
			return
		}
	}

	_, err = db.Exec("CREATE TABLE schema_version (version text)")
	if err != nil {
		_ = tx.Rollback()
		return
	}

	_, err = db.Exec("INSERT INTO schema_version VALUES (?)",
		schema.version)
	if err != nil {
		_ = tx.Rollback()
		return
	}

	tx.Commit()
	return
}

func checkSchema(db *sql.DB, schema *schema) (inabilities map[string]bool, err error) {
	row := db.QueryRow("SELECT version FROM schema_version")
	var version string
	err = row.Scan(&version)
	if err != nil {
		return
	}

	inabilities = make(map[string]bool)

	// If this is the exact version, use it, with no inabilities.
	if version == schema.version {
		return
	}

	// Otherwise, check for old version and see what we support.
	for _, compat := range schema.compats {
		if version == compat.version {
			for _, ina := range compat.inabilities {
				inabilities[ina] = true
			}

			return
		}
	}

	err = errors.New("Schema version mismatch, expect: " +
		schema.version + " got: " + version)
	return
}
