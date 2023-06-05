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

package mcutil

import (
	"context"

	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
)

const maxAssetsPerFrame = 1000

type AssetUploader struct {
	mc     *migrationcenter.Client
	source SourcePath
	frames *migrationcenterpb.Frames
}

func NewAssetUploader(mc *migrationcenter.Client, source SourcePath) *AssetUploader {
	return &AssetUploader{
		mc:     mc,
		source: source,
		frames: &migrationcenterpb.Frames{},
	}
}

func (upd *AssetUploader) Flush(ctx context.Context) error {
	if len(upd.frames.FramesData) == 0 {
		return nil
	}

	_, err := upd.mc.ReportAssetFrames(ctx, &migrationcenterpb.ReportAssetFramesRequest{
		Parent: upd.source.ProjectAndLocation.Path(),
		Source: upd.source.String(),
		Frames: upd.frames,
	})

	if err != nil {
		return err
	}

	upd.frames.FramesData = nil

	return err
}

func (upd *AssetUploader) Upload(ctx context.Context, frame *migrationcenterpb.AssetFrame) error {
	upd.frames.FramesData = append(upd.frames.FramesData, frame)
	if len(upd.frames.FramesData) == maxAssetsPerFrame {
		err := upd.Flush(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
