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

package export

import (
	"context"

	"cloud.google.com/go/bigquery"
	migrationcenter "cloud.google.com/go/migrationcenter/apiv1"
	"cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/mcutil"
	exporterschema "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/schema"
)

type MCv1 struct {
	client *migrationcenter.Client
	schema *exporterschema.ExporterSchema
}

var _ mcutil.MC = &MCv1{}

func (mc *MCv1) AssetSource(ctx context.Context, pal mcutil.ProjectAndLocation) mcutil.ObjectSource {
	it := mc.client.ListAssets(ctx, &migrationcenterpb.ListAssetsRequest{
		Parent:   pal.String(),
		PageSize: 1000,
	})
	r := newObjectReader[migrationcenterpb.Asset](it, "asset", mc.schema.AssetTable)
	src := newMigrationCenterLoadSource(r)
	return &struct {
		bigquery.LoadSource
		*objectReader[*migrationcenterpb.Asset]
	}{src, r}
}

func (mc *MCv1) GroupSource(ctx context.Context, pal mcutil.ProjectAndLocation) mcutil.ObjectSource {
	it := mc.client.ListGroups(ctx, &migrationcenterpb.ListGroupsRequest{
		Parent:   pal.String(),
		PageSize: 1000,
	})
	r := newObjectReader[migrationcenterpb.Group](it, "asset", mc.schema.GroupTable)
	src := newMigrationCenterLoadSource(r)
	return &struct {
		bigquery.LoadSource
		*objectReader[*migrationcenterpb.Group]
	}{src, r}
}

func (mc *MCv1) PreferenceSetSource(ctx context.Context, pal mcutil.ProjectAndLocation) mcutil.ObjectSource {
	it := mc.client.ListPreferenceSets(ctx, &migrationcenterpb.ListPreferenceSetsRequest{
		Parent:   pal.String(),
		PageSize: 1000,
	})
	r := newObjectReader[migrationcenterpb.PreferenceSet](it, "asset", mc.schema.PreferenceSetTable)
	src := newMigrationCenterLoadSource(r)
	return &struct {
		bigquery.LoadSource
		*objectReader[*migrationcenterpb.PreferenceSet]
	}{src, r}
}

func (mc *MCv1) AssetCount(ctx context.Context, pal mcutil.ProjectAndLocation) (int64, error) {
	resp, err := mc.client.AggregateAssetsValues(ctx, &migrationcenterpb.AggregateAssetsValuesRequest{
		Parent: pal.Path(),
		Aggregations: []*migrationcenterpb.Aggregation{
			{
				Field:               "*",
				AggregationFunction: &migrationcenterpb.Aggregation_Count_{Count: &migrationcenterpb.Aggregation_Count{}},
			},
		},
	})
	if err != nil {
		return -1, err
	}

	return resp.Results[0].GetCount().Value, nil
}
