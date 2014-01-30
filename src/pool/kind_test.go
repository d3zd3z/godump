package pool_test

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"pool"
)

func TestKind(t *testing.T) {
	k1 := pool.StringToKind("abcd")
	if k1.String() != "abcd" {
		t.Fatal("Kind mismatch")
	}

	err := quick.Check(oneKind, nil)
	if err != nil {
		t.Error(err)
	}

	err = quick.Check(oneByteKind, nil)
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
	k := pool.StringToKind(string(text))
	return k.String() == string(text)
}

func oneByteKind(text TextKind) bool {
	k := pool.BytesToKind([]byte(text))
	return bytes.Compare(k.Bytes(), []byte(text)) == 0
}
