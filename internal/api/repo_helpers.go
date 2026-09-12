package api

import (
	"database/sql"
	"errors"
)

// ignoreNoRows turns sql.ErrNoRows into a plain "not found" (nil error, zero
// value already set by the failed Scan) so callers checking existence don't
// have to special-case it themselves.
func ignoreNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}
