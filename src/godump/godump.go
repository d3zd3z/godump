package main

import "fmt"
import "pool"
import "sort"
import "time"

func main() {
	pool.IndexMain()
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
