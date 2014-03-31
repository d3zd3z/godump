package pool

import "bytes"
import "fmt"

// Utilities for generate strings made up of randomish words.

// A simple pseudorandom generator.  Designed to be fast to seed, not
// particularly good.
type rng struct {
	state uint32
}

// This is biased, but shouldn't matter here.
func (st *rng) next(limit int) int {
	st.state = (st.state * 1103515245) + 12345
	return int((st.state&0x7ffffff)>>16) % limit
}

func MakeRandomSentence(seed, size int) string {
	var buf bytes.Buffer
	var rand rng

	rand.state = uint32(seed)

	fmt.Fprintf(&buf, "%d-%d", seed, size)
	for buf.Len() < size {
		fmt.Fprintf(&buf, " %s", wordList[rand.next(len(wordList))])
	}
	buf.Truncate(size)
	return buf.String()
}

func MakeRandomChunk(size int) Chunk {
	return NewChunk("blob",
		[]byte(MakeRandomSentence(size, size)))
}

// A list of some simple words.
var wordList = []string{
	"the", "be", "to", "of", "and", "a", "in", "that", "have", "I",
	"it", "for", "not", "on", "with", "he", "as", "you", "do", "at",
	"this", "but", "his", "by", "from", "they", "we", "say", "her",
	"she", "or", "an", "will", "my", "one", "all", "would", "there",
	"their", "what", "so", "up", "out", "if", "about", "who", "get",
	"which", "go", "me", "when", "make", "can", "like", "time", "no",
	"just", "him", "know", "take", "person", "into", "year", "your",
	"good", "some", "could", "them", "see", "other", "than", "then",
	"now", "look", "only", "come", "its", "over", "think", "also"}
