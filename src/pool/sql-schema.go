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

func checkSchema(db *sql.DB, schema *schema) (err error) {
	row := db.QueryRow("SELECT version FROM schema_version")
	var version string
	err = row.Scan(&version)
	if err != nil {
		return err
	}

	if version != schema.version {
		err = errors.New("Schema version mismatch, expect: " +
			schema.version + " got: " + version)
	}
	return
}
