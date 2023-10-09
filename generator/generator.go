// cpass - A minimalist CLI random password generator focusing on convenience and security.
// Copyright (c) 2023 The cpass Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package generator

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

var letterCharset = "abcdefghijkmnpqrstuvwxyz"
var digitCharset = "0123456789"
var specialCharset = "~!@#$%^&*_+[]/?<>."

type Generator struct {
	length uint32

	uppercaseCount uint32
	digitCount     uint32
	specialCount   uint32
}

func NewGenerator(length, uppercaseCount, digitCount, specialCount uint32) (*Generator, error) {
	g := &Generator{
		length: length,

		uppercaseCount: uppercaseCount,
		digitCount:     digitCount,
		specialCount:   specialCount,
	}

	if g.length > 128 {
		return nil, fmt.Errorf("exceeded the maximum length of 128")
	}

	if g.uppercaseCount+g.digitCount+g.specialCount > g.length {
		return nil, fmt.Errorf("uppercase count (%v) + digit count (%v) + special count (%v) > length (%v)", g.uppercaseCount, g.digitCount, g.specialCount, g.length)
	}

	return g, nil
}

func (g *Generator) EntropyMax() uint64 {
	// Start with one because it is possible for a character to be empty.
	possibleChars := 1 + uint64(len(letterCharset))
	if g.uppercaseCount != 0 {
		// Uppercase doubles the letter charset variety.
		possibleChars += uint64(len(letterCharset))
	}

	if g.digitCount != 0 {
		possibleChars += uint64(len(digitCharset))
	}

	if g.specialCount != 0 {
		possibleChars += uint64(len(specialCharset))
	}

	possibleCombinations := big.NewInt(0).Exp(big.NewInt(0).SetUint64(possibleChars), big.NewInt(0).SetUint64(uint64(g.length)), big.NewInt(0))

	// Subtract one to remove the assumption of an empty password.
	possibleCombinations.Sub(possibleCombinations, big.NewInt(1))

	return uint64(possibleCombinations.BitLen())
}

func (g *Generator) EntropyMin() (uint64, error) {
	possibleCombinations := big.NewInt(1)

	nonBaseCount := g.uppercaseCount + g.digitCount + g.specialCount
	if nonBaseCount > g.length {
		return 0, fmt.Errorf("non-base letter character count exceeds the total length")
	}

	addPossibleCombinationsFn := func(charset string, count uint64) {
		// Start with one because it is possible for a character to be empty.
		charsetLength := 1 + uint64(len(charset))
		possibleCombinations = possibleCombinations.Mul(possibleCombinations,
			big.NewInt(0).Exp(big.NewInt(0).SetUint64(charsetLength), big.NewInt(0).SetUint64(count), big.NewInt(0)),
		)
	}

	baseChars := g.length - nonBaseCount

	addPossibleCombinationsFn(letterCharset, uint64(baseChars))
	addPossibleCombinationsFn(letterCharset, uint64(g.uppercaseCount))
	addPossibleCombinationsFn(digitCharset, uint64(g.digitCount))
	addPossibleCombinationsFn(specialCharset, uint64(g.specialCount))

	// Subtract one to remove the assumption of an empty password.
	possibleCombinations.Sub(possibleCombinations, big.NewInt(1))

	return uint64(possibleCombinations.BitLen()), nil
}

func (g *Generator) Generate() ([]byte, error) {
	b, err := g.generateBase()
	if err != nil {
		return nil, errors.Wrap(err, "generate letter base")
	}

	err = g.applyUppercase(b)
	if err != nil {
		return nil, errors.Wrap(err, "apply uppercase")
	}

	err = g.applyDigits(b)
	if err != nil {
		return nil, errors.Wrap(err, "apply digits")
	}

	err = g.applySpecial(b)
	if err != nil {
		return nil, errors.Wrap(err, "apply special")
	}

	return b, nil
}

func (g *Generator) generateBase() ([]byte, error) {
	ret := make([]byte, g.length)

	for i := uint32(0); i < g.length; i++ {
		b, err := secureRandomChar(letterCharset)
		if err != nil {
			return nil, errors.Wrapf(err, "generate secure random letter char #%v", i)
		}

		ret[i] = b
	}

	return ret, nil
}

func (g *Generator) seekNonBaseLetterAndApply(ptr []byte, count uint32, applyFn func(byte) (byte, error)) error {
	for i := uint32(0); i < count; i++ {
		// Limiting the search to 10k chars. This is mostly a band-aid, but
		// without it, there is a risk of deadlock.
		var ok bool

		for ii := 0; ii < 100000 && !ok; ii++ {
			pos, err := rand.Int(rand.Reader, big.NewInt(0).SetUint64(uint64(g.length)))
			if err != nil {
				return errors.Wrapf(err, "generate random pos for uppercase char #%v", i)
			}

			char := ptr[pos.Uint64()]

			if !strings.Contains(letterCharset, string(char)) {
				continue
			}

			newChar, err := applyFn(char)
			if err != nil {
				return errors.Wrap(err, "call apply func")
			}

			ptr[pos.Uint64()] = newChar
			ok = true
		}

		if !ok {
			return fmt.Errorf("bug: anti-deadlock code reached: exceeded the maximum amount of attempts looking for a free character")
		}
	}

	return nil
}

func (g *Generator) applyUppercase(ptr []byte) error {
	return g.seekNonBaseLetterAndApply(ptr, g.uppercaseCount, func(b byte) (byte, error) {
		return byte(unicode.ToUpper(rune(b))), nil
	})
}

func (g *Generator) applyDigits(ptr []byte) error {
	return g.seekNonBaseLetterAndApply(ptr, g.digitCount, func(b byte) (byte, error) {
		c, err := secureRandomChar(digitCharset)
		if err != nil {
			return 0, errors.Wrap(err, "generate secure random digit char")
		}

		return c, nil
	})
}

func (g *Generator) applySpecial(ptr []byte) error {
	return g.seekNonBaseLetterAndApply(ptr, g.specialCount, func(b byte) (byte, error) {
		c, err := secureRandomChar(specialCharset)
		if err != nil {
			return 0, errors.Wrap(err, "generate secure random special char")
		}

		return c, nil
	})
}

func secureRandomChar(charset string) (byte, error) {
	b, err := secureRandomByte()
	if err != nil {
		return 0, errors.Wrap(err, "get secure random byte")
	}

	return charset[b%byte(len(charset))], nil
}

func secureRandomByte() (byte, error) {
	bufLen, err := rand.Int(rand.Reader, big.NewInt(1024))
	if err != nil {
		return 0, errors.Wrap(err, "random-read buffer length")
	}

	b := make([]byte, bufLen.Uint64())

	_, err = rand.Read(b)
	if err != nil {
		return 0, errors.Wrap(err, "random-read")
	}

	h := sha512.Sum512(b)

	pos := h[5] % byte(len(h))
	return h[pos], nil
}
