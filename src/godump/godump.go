package main

import (
	"flag"
	"fmt"
	"log"
	"pool"
	"sort"
	"time"

	"godump/listing"
	"godump/restore"
	"meter"
)

func mainz() {
	pool.IndexMain()
}

func main() {
	flag.Parse()
	meter.Setup()
	defer meter.Shutdown()

	if flag.NArg() < 1 {
		flag.PrintDefaults()
		log.Printf("Must specify subcommand")
		return
	}
	switch flag.Arg(0) {
	case "create":
		if flag.NArg() != 2 {
			log.Printf("usage: godump create path")
			return
		}
		err := pool.CreateSqlPool(flag.Arg(1))
		if err != nil {
			log.Printf("Error creating pool: %s", err)
			return
		}

	case "list":
		if flag.NArg() != 2 {
			log.Printf("usage: godump list path")
			return
		}
		pl, err := pool.OpenPool(flag.Arg(1))
		if err != nil {
			log.Printf("Error opening pool: %s", err)
			return
		}
		defer pl.Close()
		err = listing.Run(pl)
		if err != nil {
			log.Printf("Error listing pool: %s", err)
			return
		}

	case "restore":
		if flag.NArg() != 4 {
			log.Printf("usage: godump restore path hash dir")
			return
		}
		pl, err := pool.OpenPool(flag.Arg(1))
		if err != nil {
			log.Printf("Error opening pool: %s", err)
			return
		}
		defer pl.Close()
		id, err := pool.ParseOID(flag.Arg(2))
		if err != nil {
			log.Printf("Invalid hash: %s", err)
			return
		}
		err = restore.Run(pl, id, flag.Arg(3))
		if err != nil {
			log.Printf("Error restoring backup: %s", err)
			return
		}

	default:
		log.Printf("Unknown subcommand: %s", flag.Arg(0))
		return
	}
}

func mainy() {
	// Testing hashes for size and speed.
	m := make(map[pool.OID]int)

	fmt.Printf("Building\n")
	for i := 0; i < 1000000; i++ {
		m[*pool.IntOID(i)] = i
	}

	fmt.Printf("Extracting keys\n")
	keys := make([]pool.OID, 0, len(m))

	for k, _ := range m {
		keys = append(keys, k)
	}

	fmt.Printf("Sorting\n")
	sort.Sort(OIDSlice(keys))

	for _, k := range keys {
		fmt.Printf("%s: %d\n", k.String(), m[k])
	}

	fmt.Printf("Waiting\n")
	for {
		time.Sleep(time.Second)
		fmt.Printf(".")
	}

	// index_main()
	// indexFileMain()
	// pool.PoolMain()
}

type OIDSlice []pool.OID

func (p OIDSlice) Len() int           { return len(p) }
func (p OIDSlice) Less(i, j int) bool { return p[i].Compare(&p[j]) < 0 }
func (p OIDSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
