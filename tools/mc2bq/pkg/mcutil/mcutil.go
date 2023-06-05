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
	"path"

	"cloud.google.com/go/bigquery"
)

type ProjectAndLocation struct {
	Project  string
	Location string
}

func (p *ProjectAndLocation) String() string {
	return p.Path()
}

func (p *ProjectAndLocation) Path() string {
	return path.Join("projects", p.Project, "locations", p.Location)
}

type SourcePath struct {
	ProjectAndLocation ProjectAndLocation
	SourceID           string
}

func (p *SourcePath) String() string {
	return path.Join(p.ProjectAndLocation.String(), "sources", p.SourceID)
}

// ObjectSource is a source of objects that can be
type ObjectSource interface {
	bigquery.LoadSource

	ObjectsRead() uint64
	BytesRead() uint64
}

type MC interface {
	AssetCount(ctx context.Context, pal ProjectAndLocation) (int64, error)
	AssetSource(ctx context.Context, pal ProjectAndLocation) ObjectSource
	GroupSource(ctx context.Context, pal ProjectAndLocation) ObjectSource
	PreferenceSetSource(ctx context.Context, pal ProjectAndLocation) ObjectSource
}
