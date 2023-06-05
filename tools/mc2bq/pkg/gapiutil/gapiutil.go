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

package gapiutil

import (
	"errors"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/backoff"
	"google.golang.org/api/googleapi"
)

// DefaultBackoff is the recommended backoff as documented in https://cloud.google.com/apis/design/errors#retrying_errors
var DefaultBackoff = backoff.Backoff{
	Duration: 1 * time.Second,
	Factor:   1.2,
	Jitter:   0.5,
	Cap:      10 * time.Second,
}

func IsErrorWithCode(err error, code int) bool {
	if err == nil {
		return false
	}

	var gapiError *googleapi.Error
	if !errors.As(err, &gapiError) {
		// it's not a GAPI error
		return false
	}

	return gapiError.Code == code
}

func IgnoreErrorWithCode(err error, code int) error {
	if IsErrorWithCode(err, code) {
		return nil
	}

	return err
}

// IsTransientError checks if a result of a GAPI call is a transient error.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	var gapiError *googleapi.Error
	if !errors.As(err, &gapiError) {
		return false
	}

	// See https://cloud.google.com/storage/docs/xml-api/reference-status
	switch gapiError.Code {
	case http.StatusTooManyRequests,
		http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}

	return false
}

// RetryGAPI calls fn with opts, it retries on transient network errors
// but not on GAPI errors
func RetryGAPI[T any](
	backoff backoff.Backoff,
	fn func(opts ...googleapi.CallOption) (T, error),
	opts ...googleapi.CallOption,
) (T, error) {
	var r T
	var err error
	for {
		r, err = fn(opts...)
		if IsTransientError(err) {
			time.Sleep(backoff.Step())
			continue
		}

		return r, err
	}
}
