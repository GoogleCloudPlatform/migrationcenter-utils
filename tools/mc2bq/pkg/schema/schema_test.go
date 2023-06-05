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
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/test/golden"
)

// TestObjectSerializer tests all the features of the serializer.
// The tests uses a golden output file.
// To update it run: go test -test.generate-golden-files $PWD
func TestObjectSerializer(t *testing.T) {
	type TestInnerStruct struct {
		FieldA string
		FieldB int
	}
	innerStructSchema := bigquery.Schema{
		{Name: "field_a", Type: bigquery.StringFieldType},
		{Name: "field_b", Type: bigquery.IntegerFieldType},
	}
	type TestStruct struct {
		IntScalar       int
		StringScalar    string
		BoolScalar      bool
		Float32Scalar   float32
		Float64Scalar   float64
		IntArray        []int
		Timestamp       timestamppb.Timestamp
		TimestampPtr    *timestamppb.Timestamp
		DynamicMap      map[string]string
		Record          TestInnerStruct
		RecordPtr       *TestInnerStruct
		NullPtr         *TestInnerStruct
		RecordArray     []*TestInnerStruct
		RecordMap       map[string]*TestInnerStruct
		DeprecatedField string
	}
	schema := bigquery.Schema{
		{Name: "int_scalar", Type: bigquery.IntegerFieldType},
		{Name: "string_scalar", Type: bigquery.StringFieldType},
		{Name: "bool_scalar", Type: bigquery.BooleanFieldType},
		{Name: "float32_scalar", Type: bigquery.FloatFieldType},
		{Name: "float64_scalar", Type: bigquery.FloatFieldType},
		{Name: "int_array", Type: bigquery.IntegerFieldType, Repeated: true},
		{Name: "timestamp", Type: bigquery.TimestampFieldType},
		{Name: "timestamp_ptr", Type: bigquery.TimestampFieldType},
		{Name: "dynamic_map", Type: bigquery.RecordFieldType, Repeated: true, Schema: bigquery.Schema{
			{Name: "key", Type: bigquery.StringFieldType},
			{Name: "value", Type: bigquery.StringFieldType},
		}},
		{Name: "record", Type: bigquery.RecordFieldType, Schema: innerStructSchema},
		{Name: "record_ptr", Type: bigquery.RecordFieldType, Schema: innerStructSchema},
		{Name: "record_array", Type: bigquery.RecordFieldType, Schema: innerStructSchema, Repeated: true},
		{Name: "record_map", Type: bigquery.RecordFieldType, Repeated: true, Schema: bigquery.Schema{
			{Name: "key", Type: bigquery.StringFieldType},
			{Name: "value", Type: bigquery.RecordFieldType, Schema: innerStructSchema},
		}},
		// DeprecatedField doesn't appear in the schema as it has been deprecated
	}
	// This ensures that we are testing all the field types that exist in the schema
	ensureTypeSetCoverage(t, schema)

	obj := &TestStruct{
		IntScalar:       3,
		StringScalar:    "foo",
		BoolScalar:      false,
		IntArray:        []int{1, 2, 3},
		Float32Scalar:   0.32,
		Float64Scalar:   0.64,
		DynamicMap:      map[string]string{"foo": "bar", "fizz": "buzz"},
		Timestamp:       *timestamppb.New(time.Unix(1337, 1773)),
		TimestampPtr:    timestamppb.New(time.Unix(1337, 1773)),
		Record:          TestInnerStruct{FieldA: "test", FieldB: 10},
		RecordPtr:       &TestInnerStruct{FieldA: "test", FieldB: 10},
		NullPtr:         nil,
		RecordArray:     []*TestInnerStruct{{FieldA: "1", FieldB: 1}, {FieldA: "2", FieldB: 2}},
		RecordMap:       map[string]*TestInnerStruct{"key": {FieldA: "value"}},
		DeprecatedField: "I should not be in the result JSON",
	}

	got, err := SerializeObjectToBigQuery(obj, "object", schema)
	if err != nil {
		t.Fatalf("SerializeObjectToBigQuery(%+v, ...): unexpected error: %v", obj, err)
	}

	// We do this so that the diff is more human readable
	got = prettyPrintJSON(got)
	if diff := golden.Compare(t, "exporter.json", string(got)); diff != "" {
		t.Fatalf("SerializeObjectToBigQuery(%+v, ...): mismatch (-want, +got):\n%s", obj, diff)
	}
}

func TestInvalidTimestamp(t *testing.T) {
	type TestStruct struct {
		Timestamp int
	}
	schema := bigquery.Schema{
		{Name: "timestamp", Type: bigquery.TimestampFieldType},
	}
	obj := &TestStruct{
		Timestamp: 10,
	}

	got, err := SerializeObjectToBigQuery(obj, "object", schema)
	if err == nil {
		t.Fatalf("SerializeObjectToBigQuery(%+v, ...): succeeded unexpectedly", obj)
	}

	if diff := cmp.Diff([]uint8(nil), got); diff != "" {
		t.Fatalf("SerializeObjectToBigQuery(%+v, ...): mismatch (-want, +got):\n%s", obj, diff)
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
