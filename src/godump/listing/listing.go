// Source listing.

package listing

import (
	"fmt"
	"sort"
	"time"

	"pool"
	"store"
)

type backNode struct {
	oid   *pool.OID
	date  time.Time
	props map[string]string
}

type lister struct {
	nodes []*backNode

	store.PathTrackerImpl
	store.EmptyVisitor
}

func (this *lister) Back(root *pool.OID, date time.Time, props map[string]string) (err error) {
	bn := &backNode{
		oid:   root,
		date:  date,
		props: props}
	this.nodes = append(this.nodes, bn)
	return store.Prune
}

func (this *lister) sort() {
	sort.Sort(byDate(this.nodes))
}

func (this *lister) show() {
	for _, bn := range this.nodes {
		fmt.Printf("%s %s", bn.oid.String(),
			bn.date.Format("2006-01-02_15:04"))
		keys := make([]string, 0, len(bn.props))
		for k := range bn.props {
			if k == "hash" {
				continue
			}
			keys = append(keys, k)
		}
		for _, k := range keys {
			fmt.Printf(" %s=%s", k, bn.props[k])
		}
		fmt.Printf("\n")
	}
}

func Run(pl pool.Pool) (err error) {
	backups, err := pl.Backups()
	if err != nil {
		return
	}

	var self lister
	self.InitPath()

	fmt.Printf("Listing: %d\n", len(backups))
	for _, oid := range backups {
		err = store.Walk(pl, oid, &self)
		if err != nil {
			return
		}
	}
	self.sort()
	self.show()
	return
}

type byDate []*backNode

func (a byDate) Len() int           { return len(a) }
func (a byDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDate) Less(i, j int) bool { return a[i].date.Before(a[j].date) }
