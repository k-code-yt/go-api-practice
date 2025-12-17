package dbpostgres

import (
	"errors"

	"github.com/lib/pq"
)

var (
	ErrDuplicateCode  = "23505"
	ErrDuplicateMsg   = "duplicate key violation"
	NonExistingIntKey = -1001
	JSONParsingError  = -1002
)

func IsDuplicateKeyErr(err error) bool {
	var pgErr *pq.Error
	if err != nil {
		if errors.As(err, &pgErr) {
			return pgErr.Code == pq.ErrorCode(ErrDuplicateCode)
		}
	}
	return false
}
