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

package main

import (
	"testing"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/export"
	"github.com/google/go-cmp/cmp"
)

func defaultParamsDiffOpts() cmp.Option {
	return cmp.Options{
		// empty region means default
		cmp.FilterPath(
			func(p cmp.Path) bool {
				return p.Last().String() == ".Region"
			},
			cmp.Transformer("default_region", func(s string) string {
				if s == "" {
					return "us-central1"

				}

				return s
			}),
		),
		// ignore schema
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.Last().String() == ".Schema"
		}, cmp.Ignore()),
	}
}

func TestParseFlags(t *testing.T) {
	tCases := []struct {
		Name       string
		Env        map[string]string
		Args       []string
		WantParams export.Params
		WantErr    bool
		wantAction cliAction
	}{
		{"no-args", nil, nil, export.Params{}, false, actionExitFailure},
		{"version", nil, []string{"-version"}, export.Params{}, false, actionVersion},
		{"dumb-embedded-schema", nil, []string{"-dump-embedded-schema"}, export.Params{}, false, actionDumpSchema},
		{Name: "force in env",
			Env:  map[string]string{"MC2BQ_FORCE": "1"},
			Args: []string{"project", "dataset"},
			WantParams: export.Params{
				ProjectID:       "project",
				TargetProjectID: "project",
				DatasetID:       "dataset",
				Force:           true,
			},
			WantErr:    false,
			wantAction: actionExport,
		},
		{Name: "force",
			Env:  nil,
			Args: []string{"-force", "project", "dataset"},
			WantParams: export.Params{
				ProjectID:       "project",
				TargetProjectID: "project",
				DatasetID:       "dataset",
				Force:           true,
			},
			WantErr:    false,
			wantAction: actionExport,
		},
		{Name: "target-project in env",
			Env:  map[string]string{"MC2BQ_TARGET_PROJECT": "tgt"},
			Args: []string{"project", "dataset"},
			WantParams: export.Params{
				ProjectID:       "project",
				TargetProjectID: "tgt",
				DatasetID:       "dataset",
			},
			WantErr:    false,
			wantAction: actionExport,
		},
		{Name: "region in env",
			Env:  map[string]string{"MC2BQ_REGION": "region"},
			Args: []string{"project", "dataset"},
			WantParams: export.Params{
				ProjectID:       "project",
				TargetProjectID: "project",
				DatasetID:       "dataset",
				Region:          "region",
			},
			WantErr:    false,
			wantAction: actionExport,
		},
		{Name: "project and dataset in env",
			Env: map[string]string{
				"PROJECT":       "project",
				"MC2BQ_DATASET": "dataset",
			},
			Args: []string{},
			WantParams: export.Params{
				ProjectID:       "project",
				TargetProjectID: "project",
				DatasetID:       "dataset",
			},
			WantErr:    false,
			wantAction: actionExport,
		},
		{Name: "mc2bq project env override gcloud env",
			Env: map[string]string{
				"PROJECT":       "project",
				"MC2BQ_PROJECT": "mc2bq-project",
				"MC2BQ_DATASET": "dataset",
			},
			Args: []string{},
			WantParams: export.Params{
				ProjectID:       "mc2bq-project",
				TargetProjectID: "mc2bq-project",
				DatasetID:       "dataset",
			},
			WantErr:    false,
			wantAction: actionExport,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Name, func(t *testing.T) {
			var got export.Params
			for k, v := range tCase.Env {
				t.Setenv(k, v)
			}
			act, err := parseFlags(&got, tCase.Args)
			gotErr := err != nil
			if gotErr != tCase.WantErr {
				t.Errorf("parseFlags(%+v, %+v): unexpected error: %v", got, tCase.Args, err)
			}
			if act != tCase.wantAction {
				t.Errorf("parseFlags(%+v, %+v): unexpected action want: %q got: %q", got, tCase.Args, tCase.wantAction, act)
			}

			if act != actionExport {
				return
			}

			if diff := cmp.Diff(got, tCase.WantParams, defaultParamsDiffOpts()); diff != "" {
				t.Errorf("parseFlags(%+v, %+v): diff in params (-want, +got):\n%s", got, tCase.Args, diff)
			}
		})
	}
}
