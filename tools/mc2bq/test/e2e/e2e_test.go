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

package e2e_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/backoff"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/export"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/mcutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/schema"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/test/tcx"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/test/tres"
)

var pollBackoff = backoff.Backoff{
	Duration: 5 * time.Second,
	Factor:   1.0, // We want to backoff at a constant rate
	Jitter:   0.2,
	Cap:      5 * time.Second,
}

func TestExport(t *testing.T) {
	skipE2E(t)
	const desiredAssetCount = 10_000
	ctx := tcx.NewContext(t)
	targetProjectID := tres.ObtainTargetProject(t)
	pal := mcutil.ProjectAndLocation{
		Project:  tres.ObtainProject(t),
		Location: tres.ObtainRegion(t),
	}
	_ = obtainMCWithAssets(ctx, t, pal, desiredAssetCount)

	params := export.Params{
		ProjectID:       pal.Project,
		Region:          pal.Location,
		TargetProjectID: targetProjectID,
		DatasetID:       tres.ObtainDataset(t, targetProjectID, "dataset"),
		Schema:          &schema.EmbeddedSchema,
		UserAgentSuffix: "tests",
	}

	err := export.Export(&params)
	if err != nil {
		t.Errorf("Export(%+v) failed: %v", params, err)
	}
}

func waitForPendingFrames(ctx context.Context, mc *migrationcenter.Client, path mcutil.SourcePath) error {
	return backoff.RetryUntil(ctx, pollBackoff, func() (bool, error) {
		src, err := mc.GetSource(ctx, &migrationcenterpb.GetSourceRequest{
			Name: path.String(),
		})
		if err != nil {
			return true, fmt.Errorf("wait for pending frames: %w", err)
		}

		return src.PendingFrameCount == 0, nil
	})
}

func randTimestamp() *timestamppb.Timestamp {
	maxDate := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	return &timestamppb.Timestamp{Seconds: rand.Int63n(int64(maxDate.Unix()))}
}

func randFrame() *migrationcenterpb.AssetFrame {
	return &migrationcenterpb.AssetFrame{
		FrameData: &migrationcenterpb.AssetFrame_MachineDetails{
			MachineDetails: &migrationcenterpb.MachineDetails{
				CoreCount:   rand.Int31n(10),
				CreateTime:  randTimestamp(),
				Uuid:        uuid.New().String(),
				MachineName: "Ploni",
			},
		},
	}
}

func obtainMCWithAssets(ctx context.Context, t testing.TB, pal mcutil.ProjectAndLocation, desiredAssetCount int64) *migrationcenter.Client {
	client, err := migrationcenter.NewClient(ctx)
	if err != nil {
		t.Fatalf("create mc client: %v", err)
	}

	srcPath := tres.ObtainMCSource(ctx, t, client, pal)
	t.Logf("generating frames")
	uploader := mcutil.NewAssetUploader(client, srcPath)

	for i := int64(0); i < desiredAssetCount; i++ {
		err := uploader.Upload(ctx, randFrame())
		if err != nil {
			t.Fatalf("upload assets: %v", err)
		}
		if i%1000 == 0 {
			t.Logf("Uploaded %d out of %d assets", i, desiredAssetCount)
		}
	}

	err = uploader.Flush(ctx)
	if err != nil {
		t.Fatalf("upload assets: %v", err)
	}
	t.Logf("Finished uploading, waiting for frames to be processed")
	err = waitForPendingFrames(ctx, client, srcPath)
	if err != nil {
		t.Fatalf("waiting for pending frames: %v", err)
	}

	return client
}

func skipE2E(t testing.TB) {
	const envKey = "SKIP_E2E"
	if os.Getenv(envKey) != "" {
		t.Skipf("Skipped because env contained %q", envKey)
	}
}
