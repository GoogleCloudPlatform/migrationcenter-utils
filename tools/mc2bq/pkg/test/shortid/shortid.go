// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package shortid provides a short ID generator to be used for testing.
package shortid

import (
	"math/rand"
)

// alphabet that can be part of the ID, only uses lowercase to ease
// communication of IDs between people (e.g. as in over the phone)
var alphabet = []byte{
	'1', '2', '3', '4', '5', '6', '7', '8', '9', '0',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
	'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't',
	'u', 'v', 'w', 'x', 'y', 'z',
}

const idSize = 8

func New() string {
	var buf [idSize]byte
	for i := 0; i < len(buf); i++ {
		buf[i] = alphabet[rand.Intn(len(alphabet))]
	}

	return string(buf[:])
}
