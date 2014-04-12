package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"pool"
	"sort"
	"strings"
	"time"

	"godump/cachecmd"
	"godump/config"
	"godump/dump"
	"godump/listing"
	"godump/manager"
	"godump/restore"
	"meter"
)

func mainz() {
	pool.IndexMain()
}

var configFile = flag.String("config", "/etc/godump.toml", "Path to config file")

func main() {
	flag.Parse()
	meter.Setup()
	defer meter.Shutdown()

	config, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Printf("config err: %q", err)
		return
	}

	args := flag.Args()

	if len(args) < 1 {
		flag.PrintDefaults()
		log.Printf("Must specify subcommand")
		return
	}
	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "create":
		if len(args) != 1 {
			log.Printf("usage: godump create path")
			return
		}
		err := pool.CreateSqlPool(args[0])
		if err != nil {
			log.Printf("Error creating pool: %s", err)
			return
		}

	case "list":
		if len(args) != 1 {
			log.Printf("usage: godump list path")
			return
		}
		pl, err := pool.OpenPool(args[0])
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
		if len(args) != 3 {
			log.Printf("usage: godump restore path hash dir")
			return
		}
		pl, err := pool.OpenPool(args[0])
		if err != nil {
			log.Printf("Error opening pool: %s", err)
			return
		}
		defer pl.Close()
		id, err := pool.ParseOID(args[1])
		if err != nil {
			log.Printf("Invalid hash: %s", err)
			return
		}
		err = restore.Run(pl, id, args[2])
		if err != nil {
			log.Printf("Error restoring backup: %s", err)
			return
		}

	case "dump":
		if len(args) < 3 {
			log.Printf("usage: godump dump pool dir fs=name host=name ...")
			return
		}
		pl, err := pool.OpenPool(args[0])
		if err != nil {
			log.Printf("Error opening pool: %s", err)
			return
		}
		defer pl.Close()
		path := args[1]
		props, err := encodeProps(args[2:])
		if err != nil {
			return
		}
		err = dump.Run(pl, path, props)
		if err != nil {
			log.Printf("Error backing up: %s", err)
			return
		}

	case "cache":
		err := cachecmd.Run(args)
		if err != nil {
			log.Printf("Error with cache command: %s", err)
			return
		}

	case "managed":
		err := manager.Run(config, args)
		if err != nil {
			log.Printf("Error running manager: %s", err)
		}

	default:
		log.Printf("Unknown subcommand: %s", flag.Arg(0))
		return
	}
}

// Encode the given arguments as properties.
func encodeProps(args []string) (props map[string]string, err error) {
	props = make(map[string]string)

	for _, arg := range args {
		pairs := strings.SplitN(arg, "=", 2)
		if len(pairs) != 2 {
			err = errors.New("Argument must be key=val")
			return
		}
		props[pairs[0]] = pairs[1]
	}
	return
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
