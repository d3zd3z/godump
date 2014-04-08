package cachecmd

import (
	"errors"

	"pool"
)

func Run(args []string) (err error) {
	if len(args) < 1 {
		err = errors.New("Must specify subcommand for cache command")
		return
	}

	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "list":
		err = errors.New("TODO: List command")
		return

	case "regen":
		if len(args) != 2 {
			err = errors.New("usage: cache regen poolpath backuphash")
			return
		}
		var pl pool.Pool
		pl, err = pool.OpenPool(args[0])
		if err != nil {
			return
		}
		defer pl.Close()

		var id *pool.OID
		id, err = pool.ParseOID(args[1])
		if err != nil {
			return
		}
		err = regen(pl, id)

	default:
		err = errors.New("Unknown cache subcommand, expecting 'list', 'regen'")
		return
	}

	return
}
