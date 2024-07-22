/*
Copyright 2024 Google LLC All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGroups(t *testing.T) {
	testCases := []struct {
		name    string
		path    string
		want    []string
		wantErr bool
	}{
		{
			name: "short_valid",
			path: "./testdata/short_valid.csv",
			want: []string{"Roswell", "Torque 3D", "Sensorflask", "Mahout", "genie"},
		},
		{
			name: "application_header_not_in_first_column",
			path: "./testdata/header_diff_col.csv",
			want: []string{"Roswell", "Torque 3D", "Sensorflask", "Mahout", "genie"},
		},
		{
			name:    "missing_application_header",
			path:    "./testdata/missing_application_header.csv",
			wantErr: true,
		},
		{
			name: "empty_line_in_middle",
			path: "./testdata/empty_line_in_middle.csv",
			want: []string{"Roswell", "Torque 3D", "Sensorflask", "Mahout", "genie"},
		},
		{
			name:    "wrong_file_format",
			path:    "./testdata/wrong_file_format.xml",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Groups(tc.path)
			if err != nil {
				if tc.wantErr == true {
					return
				}
				t.Errorf("Failed creating groups, err: %v", err)
			}
			opts := cmp.Options{
				cmpopts.EquateEmpty(),

				cmpopts.SortSlices(func(a, b string) bool { return a < b }),
			}
			if diff := cmp.Diff(tc.want, got, opts); diff != "" {
				t.Errorf("Diff in application groups (-want +got):\n%s", diff)
			}
		})
	}
}
