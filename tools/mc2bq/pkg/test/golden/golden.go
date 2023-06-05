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

// Package golden contains functions to generate and use golden output in tests
package golden

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var generateGoldenFiles = flag.Bool("test.generate-golden-files", false, "Generate golden output for tests. This will cause the tests to fail.")

func isGenerateMode() bool {
	if *generateGoldenFiles {
		return true
	}

	if os.Getenv("GENERATE_GOLDEN_FILES") != "" {
		return true
	}

	return false
}

// Compare golden file wantName with string got returning the diff between them.
// See go-cmp's Diff function for the exact format.
//
// To create or update the golden file run the test with the environment
// variable "GENERATE_GOLDEN_FILES". This will use got as the template for the
// golden file. In that case the comparison will succeed but the test will fail
// (using t.Error) to ensure that generator runs are not considered actually
// successful by mistake.
func Compare(t testing.TB, wantName string, got string) string {
	fullPath := goldenFile(wantName)
	if isGenerateMode() {
		err := os.MkdirAll(filepath.Dir(fullPath), 0775)
		if err != nil {
			t.Fatalf("Failed to test data directory file: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(got), 0664)
		if err != nil {
			t.Fatalf("Failed to generate golden file %q: %v", wantName, err)
		}
		// We use error so that the test itself fails but we will return an
		// empty diff so that the test keeps running maybe additional
		// golden files
		t.Error("Generated golden files so run is invalid")
	}

	wantRaw, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("could not read golden file %q: %v", wantName, err)
	}
	want := string(wantRaw)

	return cmp.Diff(want, got)
}

func goldenFile(name string) string {
	_, f, _, _ := runtime.Caller(2)
	dir := filepath.Dir(f)
	return filepath.Join(dir, "testdata", name+".golden")
}
