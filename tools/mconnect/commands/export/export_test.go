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
package export

import (
	"bytes"
	"fmt"
	"testing"
)

type fakeCastExporter struct {
	FilePath  string
	ProjectID string
	DatasetID string
	TableID   string
	Location  string
}

func (fe *fakeCastExporter) export() error {
	if fe.FilePath == "" {
		return fmt.Errorf("file path not set")
	}
	if fe.ProjectID == "" {
		return fmt.Errorf("project not set")
	}

	if fe.DatasetID == "" {
		return fmt.Errorf("dataset not set")
	}

	if fe.TableID == "" {
		return fmt.Errorf("table id not set")
	}

	if fe.Location == "" {
		return fmt.Errorf("location not set")
	}
	return nil
}

func (fe *fakeCastExporter) build(filePath, projectID, datasetID, tableID, location string) exporter {
	fe.FilePath = filePath
	fe.ProjectID = projectID
	fe.DatasetID = datasetID
	fe.TableID = tableID
	fe.Location = location
	return fe
}

type fakeMCExporter struct {
	MCProjectID     string
	MCRegion        string
	TargetProjectID string
	DatasetID       string
	Force           bool
}

func (me *fakeMCExporter) build(mcProjectID, mcRegion, targetProjectID, datasetID string, force bool) exporter {
	me.MCProjectID = mcProjectID
	me.MCRegion = mcRegion
	me.TargetProjectID = targetProjectID
	me.DatasetID = datasetID
	me.Force = force
	return me
}

func (me *fakeMCExporter) export() error {
	if me.MCProjectID == "" {
		return fmt.Errorf("file path not set")
	}

	if me.MCRegion == "" {
		return fmt.Errorf("project not set")
	}

	if me.DatasetID == "" {
		return fmt.Errorf("dataset not set")
	}

	if me.TargetProjectID == "" {
		return fmt.Errorf("table id not set")
	}

	return nil
}

func TestExportCommandArgs(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		projectID   string
		region      string
		mcProjectID string
		mcRegion    string
		wantErr     bool
	}{
		{
			name:      "valid_args",
			path:      "./report.csv",
			projectID: "123",
			region:    "us-central1",
			wantErr:   false,
		},
		{
			name:      "no_path",
			projectID: "123",
			region:    "us-central1",
			wantErr:   true,
		},
		{
			name:    "no_project",
			path:    "./report.csv",
			region:  "us-central1",
			wantErr: true,
		},
		{
			name:      "no_region",
			path:      "./report.csv",
			projectID: "123",
			wantErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := new(bytes.Buffer)
			ce := &fakeCastExporter{}
			me := &fakeMCExporter{}
			exportCmd := newExportCmd(ce, me)
			exportCmd.SetOut(actual)
			exportCmd.SetErr(actual)
			setExportFlags(exportCmd)

			exportCmd.SetArgs(args(tc.path, tc.projectID, tc.region, "", tc.mcProjectID, tc.mcRegion, false))
			err := exportCmd.Execute()

			if err != nil && tc.wantErr != true {
				t.Errorf("failed test: '%v', wantErr: %v, gotErr: %v", tc.name, tc.wantErr, err)
			}

			if err == nil && tc.wantErr == true {
				t.Errorf("failed test: '%v', wantErr: %v, gotErr: %v", tc.name, tc.wantErr, err)
			}
		})
	}
}

func TestExportCommandFlow(t *testing.T) {
	testCases := []struct {
		name            string
		path            string
		projectID       string
		region          string
		datasetID       string
		mcProjectID     string
		mcRegion        string
		force           bool
		wantPath        string
		wantProjectID   string
		wantDatasetID   string
		wantRegion      string
		wantMCProjectID string
		wantMCRegion    string
		wantForce       bool
		wantErr         bool
	}{
		{
			name:            "basic_export",
			path:            "./report.csv",
			projectID:       "123",
			region:          "us-central1",
			wantPath:        "./report.csv",
			wantProjectID:   "123",
			wantRegion:      "us-central1",
			wantMCProjectID: "123",
			wantMCRegion:    "us-central1",
			wantDatasetID:   defaultDatasetID,
			wantErr:         false,
		},
		{
			name:            "non_default_dataset",
			path:            "./report.csv",
			projectID:       "123",
			datasetID:       "abc",
			region:          "us-central1",
			wantPath:        "./report.csv",
			wantProjectID:   "123",
			wantRegion:      "us-central1",
			wantMCProjectID: "123",
			wantMCRegion:    "us-central1",
			wantDatasetID:   "abc",
			wantErr:         false,
		},
		{
			name:            "diff_cast_mc_project_id",
			path:            "./report.csv",
			projectID:       "123",
			region:          "us-central1",
			mcProjectID:     "456",
			wantPath:        "./report.csv",
			wantProjectID:   "123",
			wantRegion:      "us-central1",
			wantMCProjectID: "456",
			wantMCRegion:    "us-central1",
			wantDatasetID:   defaultDatasetID,
			wantErr:         false,
		},
		{
			name:            "diff_cast_mc_region",
			path:            "./report.csv",
			projectID:       "123",
			region:          "us-central1",
			mcRegion:        "europe-west1",
			wantPath:        "./report.csv",
			wantProjectID:   "123",
			wantRegion:      "us-central1",
			wantMCProjectID: "123",
			wantMCRegion:    "europe-west1",
			wantDatasetID:   defaultDatasetID,
			wantErr:         false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up for the export cmd
			actual := new(bytes.Buffer)
			ce := &fakeCastExporter{}
			me := &fakeMCExporter{}
			exportCmd := newExportCmd(ce, me)
			exportCmd.SetOut(actual)
			exportCmd.SetErr(actual)
			setExportFlags(exportCmd)

			exportCmd.SetArgs(args(tc.path, tc.projectID, tc.region, tc.datasetID, tc.mcProjectID, tc.mcRegion, tc.force))
			err := exportCmd.Execute()

			if err != nil && tc.wantErr != true {
				t.Errorf("failed test: '%v', err: %v", tc.name, err)
			}

			if err := ce.diffVars(tc.wantPath, tc.wantProjectID, tc.wantDatasetID, castTableID, tc.wantRegion); err != nil {
				t.Errorf("failed test: '%v', fakeCastExporter diff, err: %v", tc.name, err)
			}

			if err := me.diffVars(tc.wantMCProjectID, tc.wantMCRegion, tc.wantProjectID, tc.wantDatasetID, tc.wantForce); err != nil {
				t.Errorf("failed test: '%v', fakeMCExporter diff, err: %v", tc.name, err)
			}

		})
	}

}

func args(path, project, region, dataset, mcProject, mcRegion string, force bool) []string {
	var args []string

	if path != "" {
		args = append(args, "--path="+path)
	}
	if project != "" {
		args = append(args, "--project="+project)
	}
	if region != "" {
		args = append(args, "--region="+region)
	}
	if dataset != "" {
		args = append(args, "--dataset="+dataset)
	}
	if mcProject != "" {
		args = append(args, "--mc-project="+mcProject)
	}
	if mcRegion != "" {
		args = append(args, "--mc-region="+mcRegion)
	}
	if force {
		args = append(args, "--force")
	}
	return args
}

// diffVars checks if the wanted parameters equal fakeCastExporter's vars.
// Returns an error upon inequality, otherwise nil.
func (ce *fakeCastExporter) diffVars(wantFilePath, wantProjectID, wantDatasetID, wantTableID, wantLocation string) error {
	if ce.FilePath != wantFilePath {
		return fmt.Errorf("filePath doesn't match, want: '%v', got: '%v'", wantFilePath, ce.FilePath)
	}
	if ce.ProjectID != wantProjectID {
		return fmt.Errorf("projectID doesn't match, want: '%v', got: '%v'", wantProjectID, ce.ProjectID)
	}
	if ce.DatasetID != wantDatasetID {
		return fmt.Errorf("datasetID doesn't match, want: '%v', got: '%v'", wantDatasetID, ce.DatasetID)
	}
	if ce.TableID != wantTableID {
		return fmt.Errorf("tableID doesn't match, want: '%v', got: '%v'", wantTableID, ce.TableID)
	}
	if ce.Location != wantLocation {
		return fmt.Errorf("Location doesn't match, want: '%v', got: '%v'", wantLocation, ce.Location)
	}
	return nil
}

// diffVars checks if the wanted parameters equal fakeMCExporter's vars.
// Returns an error upon inequality, otherwise nil.
func (me *fakeMCExporter) diffVars(wantMCProjectID, wantMCRegion, wantTargetProjectID, wantDatasetID string, wantForce bool) error {
	if me.MCProjectID != wantMCProjectID {
		return fmt.Errorf("mcProjectID doesn't match, want: '%v', got: '%v'", wantMCProjectID, me.MCProjectID)
	}
	if me.MCRegion != wantMCRegion {
		return fmt.Errorf("projectID doesn't match, want: '%v', got: '%v'", wantMCRegion, me.MCRegion)
	}
	if me.TargetProjectID != wantTargetProjectID {
		return fmt.Errorf("targetProjectID doesn't match, want: '%v', got: '%v'", wantTargetProjectID, me.TargetProjectID)
	}
	if me.DatasetID != wantDatasetID {
		return fmt.Errorf("datasetID doesn't match, want: '%v', got: '%v'", wantDatasetID, me.DatasetID)
	}
	if me.Force != wantForce {
		return fmt.Errorf("force doesn't match, want: '%v', got: '%v'", wantForce, me.Force)
	}
	return nil
}
