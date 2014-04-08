package cachecmd

import (
	"errors"

	"cache"
	"pool"
)

// Run an expiration on the given pool.
func expire(pl pool.Pool) (err error) {
	tx := pool.GetSql(pl)
	if tx == nil {
		err = errors.New("Pool type doesn't contain SQL database")
		return
	}

	err = cache.RunExpire(tx)
	if err != nil {
		return
	}

	err = pl.Flush()
	return
}
