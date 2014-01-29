package pool_test

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"pool"
)

func TestKind(t *testing.T) {
	k1 := pool.NewKind("abcd")
	if k1.String() != "abcd" {
		t.Fatal("Kind mismatch")
	}

	err := quick.Check(oneKind, nil)
	if err != nil {
		t.Error(err)
	}
}

// Use quick to test lots of variants.
type TextKind string

func (k TextKind) Generate(r *rand.Rand, _ int) reflect.Value {
	word := make([]byte, 4)
	for i := 0; i < 4; i++ {
		word[i] = byte(rand.Intn(0x100))
	}
	return reflect.ValueOf(TextKind(word))
}

func oneKind(text TextKind) bool {
	k := pool.NewKind(string(text))
	return k.String() == string(text)
}
