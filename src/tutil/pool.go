package tutil

// Tests that want a pool.
import (
	"testing"

	"pool"
)

type PoolTest struct {
	T    *testing.T
	Tmp  *TempDir
	Pool pool.Pool
}

func NewPoolTest(t *testing.T) (pt *PoolTest) {
	var self PoolTest
	self.T = t
	self.Tmp = NewTempDir(t)

	base := self.Tmp.Path() + "/pool"
	err := pool.CreateSqlPool(base)
	if err != nil {
		t.Errorf("Unable to create pool: '%s'", err)
	}

	self.Pool, err = pool.OpenPool(base)
	if err != nil {
		t.Errorf("Unable to open created pool: '%s'", err)
	}

	return &self
}

func (self *PoolTest) Clean() {
	if self.Pool != nil {
		self.Pool.Close()
		self.Pool = nil
	}
	self.Tmp.Clean()
}
