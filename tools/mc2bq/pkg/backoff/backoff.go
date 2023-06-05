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

// Package backoff implements exponential backoff functionality
package backoff

import (
	"math/rand"
	"time"
)

// DefaultBackoff is a good default backoff configuration for interacting with
// GCP APIs
var DefaultBackoff Backoff = Backoff{
	Duration: 10 * time.Millisecond,
	Factor:   4.0,
	Jitter:   0.2,
	Cap:      10 * time.Second,
}

// Backoff is an exponential backoff duration step tracker
type Backoff struct {
	// Initial duration
	Duration time.Duration
	// Factor of the exponential backoff, must be more than 1
	Factor float64
	// Jitter factor, some duration between 0 and Jutter*Duration will be added, 0 means no jitter
	Jitter float64
	// Maximum duration to wait, 0 means no cap
	Cap time.Duration
}

// Step returns the current recommended time to wait, modifies backoff to the next
// step
func (b *Backoff) Step() time.Duration {
	sleep, next := delay(b.Duration, b.Cap, b.Factor, b.Jitter)
	b.Duration = next
	return sleep
}

// delay implements the core delay algorithm used in this package.
func delay(duration, cap time.Duration, factor, jitter float64) (sleep time.Duration, next time.Duration) {
	sleep = duration
	// add jitter for this step
	if jitter > 0 {
		sleep = duration + time.Duration(rand.Float64()*jitter*float64(duration))
	}

	// calculate next duration
	if factor > 1 {
		duration = time.Duration(float64(duration) * factor)
		if cap > 0 && duration > cap {
			duration = cap
		}
	}

	return sleep, duration
}
