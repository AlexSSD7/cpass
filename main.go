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

package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/AlexSSD7/cpass/generator"
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

const Version = "v0.1.0"

func askUint32(stdinReader *bufio.Reader, prompt string) (uint32, error) {
	fmt.Printf("%s > ", prompt)
	b, err := stdinReader.ReadBytes('\n')
	if err != nil {
		return 0, errors.Wrap(err, "read bytes")
	}

	v, err := strconv.ParseUint(strings.ReplaceAll(string(b), "\n", ""), 10, 32)
	if err != nil {
		return 0, errors.Wrap(err, "parse uint")
	}

	return uint32(v), nil
}

func askYesNo(stdinReader *bufio.Reader, prompt string) (bool, error) {
	fmt.Printf("%s [y/n] > ", prompt)
	b, err := stdinReader.ReadBytes('\n')
	if err != nil {
		return false, errors.Wrap(err, "read bytes")
	}

	return strings.EqualFold(string(b[0]), "y"), nil
}

func isPowerOfTwo[T constraints.Unsigned](v T) bool {
	return v > 0 && (v&(v-1)) == 0
}

func main() {
	fmt.Printf("cpass %v %v/%v %v. Copyright (c) 2023 The cpass Authors. Distributed under GNU GPL v3, this program comes with ABSOLUTELY NO WARRANTY.\n", Version, runtime.GOOS, runtime.GOARCH, runtime.Version())

	stdinReader := bufio.NewReader(os.Stdin)

	var pwLen uint32
	var err error

	for {
		pwLen, err = askUint32(stdinReader, "Password length")
		if err != nil {
			fmt.Printf("Error: ask for password length: %s\n", err)
			os.Exit(1)
		}

		if pwLen%10 == 0 || isPowerOfTwo(pwLen) {
			fmt.Print("WARN: Detected a common base-ten (10, 20, etc) or power-of-two (16, 32, etc) password length. It's recommended to use something more random.\n")
			yes, err := askYesNo(stdinReader, "Change password length?")
			if err != nil {
				fmt.Printf("Error: ask for yes/no: %s\n", err)
				os.Exit(1)
			}

			if !yes {
				fmt.Print("WARN: Going with unsafe password length.\n")
				break
			}
		} else {
			break
		}
	}

	uppercaseCount, err := askUint32(stdinReader, "Number of uppercase characters to include (ABCDE)")
	if err != nil {
		fmt.Printf("Error: ask for uppercase character count: %s\n", err)
		os.Exit(1)
	}

	digitCount, err := askUint32(stdinReader, "Number of digit characters to include (01234)")
	if err != nil {
		fmt.Printf("Error: ask for digit character count: %s\n", err)
		os.Exit(1)
	}

	specialCount, err := askUint32(stdinReader, "Number of special characters to include (~!@#$)")
	if err != nil {
		fmt.Printf("Error: ask for special character count: %s\n", err)
		os.Exit(1)
	}

	g, err := generator.NewGenerator(pwLen, uppercaseCount, digitCount, specialCount)
	if err != nil {
		fmt.Printf("Error: create password generator instance: %s\n", err)
		os.Exit(1)
	}

	b, err := g.Generate()
	if err != nil {
		fmt.Printf("Error: generate password: %s\n", err)
		os.Exit(1)
	}

	entropyMax := g.EntropyMax()
	entropyMin, err := g.EntropyMin()
	if err != nil {
		fmt.Printf("Error: get min entropy: %s\n", err)
		os.Exit(1)
	}

	entropyAvg := (float64(g.EntropyMax()) + float64(entropyMin)) / 2

	fmt.Printf(`
Generated Password: %v

Entropy (min/realistic/max bits): %v/%v/%v (%v)
`, string(b), entropyMin, entropyAvg, entropyMax, getRatingString(entropyAvg))

	// Clean up memory.
	{
		for i := 0; i < len(b); i++ {
			b[i] = 0
		}

		_, _ = rand.Read(b)
	}
}

func getRatingString(entropyBits float64) string {
	switch {
	case entropyBits <= 32:
		return "Very Poor"
	case entropyBits <= 48:
		return "Poor"
	case entropyBits <= 72:
		return "Weak"
	case entropyBits <= 96:
		return "Good"
	case entropyBits <= 120:
		return "Excellent"
	default:
		return "Overkill"
	}
}
