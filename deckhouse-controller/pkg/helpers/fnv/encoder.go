// Copyright 2021 Flant JSC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fnv

import (
	"encoding/base32"
	"fmt"
	"hash/fnv"
	"strings"
)

var encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567")

func Encode(input string) error {
	if input == "" {
		return fmt.Errorf("not enough arguments to encode")
	}
	toDecodeString := []byte(input)
	encodedString := strings.TrimRight(encoding.EncodeToString(fnv.New64().Sum(toDecodeString)), "=")
	fmt.Print(encodedString)

	return nil
}
