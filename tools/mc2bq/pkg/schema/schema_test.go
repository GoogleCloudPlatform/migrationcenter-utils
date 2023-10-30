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

package schema

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/migrationcenter/apiv1/migrationcenterpb"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/test/golden"
)

// TestObjectSerializer tests all the features of the serializer.
// The tests uses a golden output file.
// To update it run: go test -test.generate-golden-files $PWD
func TestObjectSerializer(t *testing.T) {
	schema := EmbeddedSchema.AssetTable
	// This ensures that we are testing all the field types that exist in the schema
	ensureTypeSetCoverage(t, schema)

	asset := migrationcenterpb.Asset{
		Name:       "foo",                              // string
		CreateTime: timestamppb.New(time.Unix(10, 10)), // timestamp
		Labels: map[string]string{ // map
			"key":  "value",
			"key2": "value2",
		},
		InsightList: &migrationcenterpb.InsightList{
			Insights: []*migrationcenterpb.Insight{
				{
					Insight: &migrationcenterpb.Insight_MigrationInsight{ // one_of
						MigrationInsight: &migrationcenterpb.MigrationInsight{
							Fit: &migrationcenterpb.FitDescriptor{
								FitLevel: migrationcenterpb.FitDescriptor_FIT, // enum
							},
						},
					},
				},
			},
		},
		AssetDetails: &migrationcenterpb.Asset_MachineDetails{
			MachineDetails: &migrationcenterpb.MachineDetails{
				MachineName: "foo",
				MemoryMb:    10, // integer
			},
		},
	}

	serializer := NewSerializer[*migrationcenterpb.Asset]("asset", schema)

	got, err := serializer(&asset)
	if err != nil {
		t.Fatalf("SerializeObjectToBigQuery(%+v, ...): unexpected error: %v", &asset, err)
	}

	// We do this so that the diff is more human readable
	got = prettyPrintJSON(got)
	if diff := golden.Compare(t, "exporter.json", string(got)); diff != "" {
		t.Fatalf("SerializeObjectToBigQuery(%+v, ...): mismatch (-want, +got):\n%s", &asset, diff)
	}
}

// TestMarshalSchema tests the MarshalJSON override
func TestMarshalSchema(t *testing.T) {
	s := ExporterSchema{
		AssetTable: bigquery.Schema{
			{Name: "Foo", Type: bigquery.IntegerFieldType},
		},
		GroupTable: bigquery.Schema{
			{Name: "Foo", Type: bigquery.StringFieldType},
		},
		PreferenceSetTable: bigquery.Schema{
			{Name: "Foo", Type: bigquery.BooleanFieldType},
		},
	}

	got, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshalling of ExporterSchema failed: %v", err)
	}
	if diff := golden.Compare(t, "marshal_schema.json", string(got)); diff != "" {
		t.Fatalf("JSON marshalling mismatch (-want, +got):\n%s", diff)
	}
}

// TestUnmarshalEmptySchema checks that an empty schema doesn't marshal.
// Having an asset schema is mandatory.
func TestUnmarshalEmptySchema(t *testing.T) {
	var s ExporterSchema
	err := json.Unmarshal([]byte{'{', '}'}, &s)
	if err == nil {
		t.Fatal("JSON unmarshalling unexpectedly succeeded")
	}
}

// TestMinimalSchema checks that a schema with only assets still works.
func TestMinimalSchema(t *testing.T) {
	want := ExporterSchema{
		AssetTable: bigquery.Schema{
			{Name: "Foo", Type: bigquery.IntegerFieldType},
		},
	}

	var got ExporterSchema

	err := json.Unmarshal([]byte(`{"asset_table":[{"name": "Foo", "type":"INTEGER"}]}`), &got)
	if err != nil {
		t.Fatalf("JSON unmarshalling failed: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Unexpected unmarshalling schema (-want, +got):\n%s", diff)
	}
}

func prettyPrintJSON(buf []byte) []byte {
	var tmp map[string]any
	err := json.Unmarshal(buf, &tmp)
	if err != nil {
		// This is a utility for testing so we can panic
		panic(err)
	}

	// Since tmp has just been unmarshalled we know it will successfully marshal
	res, _ := json.MarshalIndent(tmp, "", "  ")
	// Stops Presubmit:TerminatingNewline from complaining
	return append(res, '\n')
}

// calculateTypeSet adds the field types that are used to the schema to the foundTypes set.
func calculateTypeSet(schema bigquery.Schema, foundTypes map[bigquery.FieldType]bool) {
	for _, field := range schema {
		foundTypes[field.Type] = true
		calculateTypeSet(field.Schema, foundTypes)
	}
}

func ensureTypeSetCoverage(t testing.TB, testSchema bigquery.Schema) {
	embeddedTypeSet := map[bigquery.FieldType]bool{}
	calculateTypeSet(EmbeddedSchema.AssetTable, embeddedTypeSet)
	calculateTypeSet(EmbeddedSchema.GroupTable, embeddedTypeSet)
	calculateTypeSet(EmbeddedSchema.PreferenceSetTable, embeddedTypeSet)

	coveredTypeSet := map[bigquery.FieldType]bool{}
	calculateTypeSet(testSchema, coveredTypeSet)

	for k := range coveredTypeSet {
		delete(embeddedTypeSet, k)
	}
	if len(embeddedTypeSet) > 0 {
		var sb strings.Builder
		for k := range embeddedTypeSet {
			sb.WriteString(" ")
			sb.WriteString(string(k))
		}
		t.Errorf("types exists in embedded schema that are not covered by unit tests:%s", sb.String())
	}
}
