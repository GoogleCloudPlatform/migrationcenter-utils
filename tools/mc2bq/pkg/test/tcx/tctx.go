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

// package tcx implements utilities to create a context for a given test
package tcx

import (
	"context"
	"testing"
	"time"
)

type deadliner interface {
	Deadline() (deadline time.Time, ok bool)
}

// NewContext creates a new context for test, the context will be canceled when
// the test starts cleanup.
// Don't use this context in cleanup functions as it is cancelled.
func NewContext(t testing.TB) context.Context {
	ctx, canceFunc := context.WithCancel(context.Background())
	t.Cleanup(canceFunc)
	if d, ok := t.(deadliner); ok {
		if deadline, ok := d.Deadline(); ok {
			var cancelFunc context.CancelFunc
			ctx, cancelFunc = context.WithDeadline(ctx, deadline)
			t.Cleanup(cancelFunc)
		}
	}

	return ctx
}
