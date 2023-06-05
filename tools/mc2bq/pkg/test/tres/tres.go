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

// Package tres implements various helper methods to obtain test resources.
package tres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/backoff"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/gapiutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/mcutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/test/shortid"
	"google.golang.org/api/iterator"
)

const defaultRegion = "us-central1"

var pollBackoff = backoff.Backoff{
	Duration: 5 * time.Second,
	Factor:   1.0, // We want to backoff at a constant rate
	Jitter:   0.2,
	Cap:      5 * time.Second,
}

func ObtainProject(t testing.TB) string {
	projectID := os.Getenv("PROJECT")
	if projectID == "" {
		t.Fatal("No project defined, use the PROJECT environment variable to set the project to be used for testing")
	}

	return projectID
}

func ObtainTargetProject(t testing.TB) string {
	if id := os.Getenv("TARGET_PROJECT_ID"); id != "" {
		return id
	}

	return ObtainProject(t)
}

func ObtainRegion(t testing.TB) string {
	if region := os.Getenv("REGION"); region != "" {
		return region
	}

	return defaultRegion
}

func ObtainMCSource(ctx context.Context, t testing.TB, mc *migrationcenter.Client, path mcutil.ProjectAndLocation) mcutil.SourcePath {
	srcPath := mcutil.SourcePath{
		ProjectAndLocation: path,
		SourceID:           GenerateResourceName("source"),
	}
	err := createSource(ctx, mc, srcPath)
	if err != nil {
		t.Fatalf("create source for test: %v", err)
	}
	t.Cleanup(func() {
		t.Logf("cleaning up mc source %q...", srcPath.String())
		ctx, cancelFunc := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancelFunc()
		err := deleteSource(ctx, mc, srcPath)
		if err != nil {
			t.Logf("error cleaning up source: %v", err)
		}
	})

	t.Logf("obtaining mc source %q", srcPath.String())
	return srcPath
}

func ObtainDataset(t testing.TB, project string, prefix string) string {
	datasetID := GenerateResourceName("dataset")
	t.Logf("obtaining dataset %q", datasetID)
	t.Cleanup(func() {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancelFunc()
		t.Logf("cleaning dataset %q...", datasetID)
		backoff := gapiutil.DefaultBackoff
		bq, err := bigquery.NewClient(ctx, project)
		if err != nil {
			t.Logf("error during cleanup: %v", err)
			return
		}
		ds := bq.Dataset(datasetID)
		tbls := ds.Tables(ctx)
		for {
			tbl, err := tbls.Next()
			if errors.Is(err, iterator.Done) {
				break
			} else if err != nil {
				t.Logf("error during cleanup: %v", err)
				return
			}
			err = tbl.Delete(ctx)
			if err != nil {
				t.Logf("error during cleanup: %v", err)
				return
			}
		}
		for {
			err = ds.Delete(ctx)
			if gapiutil.IsTransientError(err) {
				time.Sleep(backoff.Step())
				continue
			} else if err != nil {
				t.Logf("error during cleanup: %v", err)
			}

			return
		}
	})

	return datasetID
}

func deleteSource(ctx context.Context, mc *migrationcenter.Client, path mcutil.SourcePath) error {
	op, err := mc.DeleteSource(ctx, &migrationcenterpb.DeleteSourceRequest{
		Name: path.String(),
	})
	if err != nil {
		return fmt.Errorf("delete source %q: %w", path.String(), err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("wait delete source %q: %w", path.String(), err)
	}

	return nil
}

func createSource(ctx context.Context, mc *migrationcenter.Client, path mcutil.SourcePath) error {
	op, err := mc.CreateSource(ctx, &migrationcenterpb.CreateSourceRequest{
		Parent:   path.ProjectAndLocation.Path(),
		SourceId: path.SourceID,
		Source: &migrationcenterpb.Source{
			Type: migrationcenterpb.Source_SOURCE_TYPE_INVENTORY_SCAN,
		},
	})
	if err != nil {
		return fmt.Errorf("create source %q: %w", path.String(), err)
	}

	_, err = op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("wait create source %q: %w", path.String(), err)
	}

	return nil
}

func GenerateResourceName(prefix string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, time.Now().UTC().Format("20060102"), shortid.New())
}
