package util

import (
	"math/rand"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomInt generates a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	// strings.Builder is a type from the strings package that collects bytes into one buffer
	// strings can't be changed, so s += "x" in a loop would copy the whole string every time
	// its zero value is ready to use, no setup needed
	var sb strings.Builder
	// number of letters in alphabet (26)
	k := len(alphabet)

	// runs n times (range over an int needs go 1.22+)
	for range n {
		// rand.Intn(k) returns 0 to k-1 (0-25), always a valid index into alphabet
		// the n in Intn is the upper limit it takes, so "an int below n"
		// indexing a string gives a byte, so c is one random letter
		c := alphabet[rand.Intn(k)]
		// add the letter to the end of the builder's buffer
		sb.WriteByte(c)
	}

	// return everything collected so far as one string
	return sb.String()
}

// RandomOwner generates a random owner name
func RandomOwner() string {
	return RandomString(7)
}

// RandomMoney generates random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

// RandomCurrency generates a random currency code
func RandomCurrency() string {
	currencies := []string{"USD", "EUR", "CAD"}
	n := len(currencies)
	return currencies[rand.Intn(n)]
}
