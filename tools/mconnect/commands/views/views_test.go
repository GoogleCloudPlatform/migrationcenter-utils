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

// Package views contains the implmentation of the create-views command used to create views in BigQuery using CAST and Migration Center data and generate a link
// to a looker studio report using that data.
package views

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

type fakeViewCreator struct {
	ProjectID string
	DatasetID string
}

func (vc *fakeViewCreator) build(projectID, datasetID string) viewCreator {
	vc.ProjectID = projectID
	vc.DatasetID = datasetID
	return vc
}

func (vc *fakeViewCreator) createView(ctx context.Context, metadata viewMetadata) error {
	if vc.ProjectID == "" {
		return fmt.Errorf("project not set")
	}

	if vc.DatasetID == "" {
		return fmt.Errorf("dataset not set")
	}
	return nil
}

func TestCreateViewsCommandArgs(t *testing.T) {
	testCases := []struct {
		name    string
		project string
		dataset string
		wantErr bool
	}{
		{
			name:    "valid_args",
			project: "123",
			dataset: "mcCast",
			wantErr: false,
		},

		{
			name:    "no_project",
			dataset: "mcCast",
			wantErr: true,
		},
		{
			name:    "no_dataset",
			project: "123",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := new(bytes.Buffer)
			vc := &fakeViewCreator{}
			viewsCmd := NewCreateViewsCmd(vc)
			viewsCmd.SetOut(actual)
			viewsCmd.SetErr(actual)
			setCreateViewsFlags(viewsCmd)

			viewsCmd.SetArgs(args(tc.project, tc.dataset, false))
			err := viewsCmd.Execute()

			if err != nil && tc.wantErr != true {
				t.Errorf("failed test: '%v', wantErr: %v, gotErr: %v", tc.name, tc.wantErr, err)
			}

			if err == nil && tc.wantErr == true {
				t.Errorf("failed test: '%v', wantErr: %v, gotErr: %v", tc.name, tc.wantErr, err)
			}
		})
	}
}

func args(project, dataset string, force bool) []string {
	var args []string

	if project != "" {
		args = append(args, "--project="+project)
	}

	if dataset != "" {
		args = append(args, "--dataset="+dataset)
	}
	if force {
		args = append(args, "--force")
	}
	return args
}
