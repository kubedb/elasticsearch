package password

import (
	"crypto/rand"
	"math/big"
)

type Charset int

const (
	Uppercase     Charset = 1 << iota // 1 << 0 which is 00000001
	Lowercase                         // 1 << 1 which is 00000010
	Numbers                           // 1 << 2 which is 00000100
	Symbols                           // 1 << 3 which is 00001000
	SimpleSymbols                     // 1 << 4 which is 00001000
)

var (
	uppercase          = []byte(`ABCDEFGHIJKLMNOPQRSTUVWXYZ`)
	len_uppercase      = len(uppercase)
	lowercase          = []byte(`abcdefghijklmnopqrstuvwxyz`)
	len_lowercase      = len(lowercase)
	numbers            = []byte(`0123456789`)
	len_numbers        = len(numbers)
	symbols            = []byte(`!"#$%&'()*+,-./:;<=>?@^[\]_{|}~` + "`")
	len_symbols        = len(symbols)
	simple_symbols     = []byte(`!#$%&*+-=?@^_|`)
	len_simple_symbols = len(simple_symbols)
)

func Generate(n int) string {
	return GenerateForCharset(n, Uppercase|Lowercase|Numbers|SimpleSymbols)
}

func GenerateForCharset(n int, chset Charset) string {
	buf := make([]byte, n)

	count := 0
	if chset&Uppercase != 0 {
		count += len_uppercase
	}
	if chset&Lowercase != 0 {
		count += len_lowercase
	}
	if chset&Numbers != 0 {
		count += len_numbers
	}
	if chset&Symbols != 0 {
		count += len_symbols
	}
	if chset&SimpleSymbols != 0 {
		count += len_simple_symbols
	}
	max := big.NewInt(int64(count))

	for i := 0; i < n; i++ {
		r, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		idx := int(r.Int64())

		if chset&Uppercase != 0 {
			if idx < len_uppercase {
				buf[i] = uppercase[idx]
				continue
			} else {
				idx -= len_uppercase
			}
		}
		if chset&Lowercase != 0 {
			if idx < len_lowercase {
				buf[i] = lowercase[idx]
				continue
			} else {
				idx -= len_lowercase
			}
		}
		if chset&Numbers != 0 {
			if idx < len_numbers {
				buf[i] = numbers[idx]
				continue
			} else {
				idx -= len_numbers
			}
		}
		if chset&Symbols != 0 {
			if idx < len_symbols {
				buf[i] = symbols[idx]
				continue
			} else {
				idx -= len_symbols
			}
		}
		if chset&SimpleSymbols != 0 {
			if idx < len_simple_symbols {
				buf[i] = simple_symbols[idx]
				continue
			} else {
				idx -= len_simple_symbols
			}
		}
	}
	return string(buf)
}
