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

// Package schema contains the latest BigQuery schemas and utilities for
// serializeing Migration Center objects to schema compatible JSON objects.
package schema

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/messages"
)

// ExporterSchema is the collection of the individual table schemas that will
// be used during export
type ExporterSchema struct {
	AssetTable         bigquery.Schema `json:"asset_table" bq:"assets"`
	GroupTable         bigquery.Schema `json:"group_table" bq:"groups"`
	PreferenceSetTable bigquery.Schema `json:"preference_set_table" bq:"preference_sets"`
}

var _ json.Marshaler = &ExporterSchema{}
var _ json.Unmarshaler = &ExporterSchema{}

// MarshalJSON implementes the json.Marshaller interface.
func (s *ExporterSchema) MarshalJSON() ([]byte, error) {
	// We need to implement the Marshaller interface because bigquery.Schema marshals with
	// ToJSONFields() insteald of implementing the Marshaller interface.

	myType := reflect.TypeOf(s).Elem()
	myValue := reflect.ValueOf(s).Elem()
	res := map[string]json.RawMessage{}
	for i := 0; i < myType.NumField(); i++ {
		key := myType.Field(i).Tag.Get("json")
		schema := myValue.Field(i).Interface().(bigquery.Schema)
		fields, err := schema.ToJSONFields()
		if err != nil {
			return nil, err
		}
		res[key] = fields

	}
	return json.Marshal(res)
}

// UnmarshalJSON implementes the json.Unmarshaller interface.
func (s *ExporterSchema) UnmarshalJSON(buf []byte) error {
	// We implement the Unmarshaller interface because bigquery.Schema doesn't
	// simply unmarshal but it is loaded by using SchemaFromJSON()

	myType := reflect.TypeOf(s).Elem()
	myValue := reflect.ValueOf(s).Elem()
	raw := map[string]json.RawMessage{}
	err := json.Unmarshal(buf, &raw)
	if err != nil {
		return err
	}
	for i := 0; i < myType.NumField(); i++ {
		key := myType.Field(i).Tag.Get("json")
		rawSchema, ok := raw[key]
		if !ok {
			// schema older than this entity
			continue
		}
		loadedSchema, err := bigquery.SchemaFromJSON(rawSchema)
		if err != nil {
			return fmt.Errorf("unmarshal %s: %w", key, err)
		}
		myValue.Field(i).Set(reflect.ValueOf(loadedSchema))
	}

	if len(s.AssetTable) == 0 {
		return messages.NewError(messages.ErrorInvalidSchema)
	}

	return nil
}

//go:embed migrationcenter_v1_latest.schema.json
var rawEmbeddedSchema []byte

// EmbeddedSchema is the embedded schema distributed with the tool.
var EmbeddedSchema ExporterSchema

func init() {
	err := json.Unmarshal(rawEmbeddedSchema, &EmbeddedSchema)
	if err != nil {
		panic(err)
	}
}

// NewSerializer creates a type safe serializer for type T.
// It's the callers responsibility to make sure that the schema and type T match.
// root describes the root node string that will appear in errors.
func NewSerializer[T any](root string, schema bigquery.Schema) func(obj T) ([]byte, error) {
	return func(obj T) ([]byte, error) {
		return SerializeObjectToBigQuery(obj, root, schema)
	}
}

// SerializeObjectToBigQuery serializes an object as a BigQuery compatible JSON.
// A '\n' is appended at the end of the json data.
// The function should never return an error in production, if it fails it's a bug
// resulting from a mismatch between the API object and the BigQuery schema and both are generated
// from the same protobuf.
func SerializeObjectToBigQuery(obj any, root string, schema bigquery.Schema) ([]byte, error) {
	serializedObj, err := normalizeToSchema(
		obj,
		&bigquery.FieldSchema{
			Name:   root,
			Schema: schema,
			Type:   bigquery.RecordFieldType,
		},
	)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(serializedObj)
	if err != nil {
		return nil, err
	}

	return append(res, '\n'), err
}

// normalizeToSchema normalizes obj to according to the provided schema.
// Specifically converts structs to maps, and maps to key-value lists.
func normalizeToSchema(obj any, schema *bigquery.FieldSchema) (any, error) {
	// short circuit for nil values
	if obj == nil {
		return nil, nil
	}

	objValue := reflect.ValueOf(obj)
	if objValue.Type().Kind() == reflect.Ptr {
		if objValue.IsNil() {
			return nil, nil
		}

		objValue = objValue.Elem()
	}
	objType := objValue.Type()
	if schema.Repeated {
		switch objType.Kind() {
		case reflect.Slice:
			// This is an array, we need to just normalize each item in the array.
			itemSchema := *schema
			itemSchema.Repeated = false // we are serializing the item so it's not repeated
			result := make([]any, objValue.Len())
			for i := 0; i < objValue.Len(); i++ {
				var err error
				result[i], err = normalizeToSchema(objValue.Index(i).Interface(), &itemSchema)
				if err != nil {
					return nil, wrapWithSerializeError(fmt.Sprintf("%s[%d]", schema.Name, i), err)
				}
			}
			return result, nil
		case reflect.Map:
			// This is a dynamic map, those are not supported by BigQuery so we need to convert it to an
			// array in the form of [struct{key: string, value: T}, ...]
			if len(schema.Schema) != 2 {
				return nil, wrapWithSerializeError(schema.Name, errors.New("schema for dynamic map is invalid"))
			}

			if schema.Schema[1].Name != "value" {
				return nil, wrapWithSerializeError(schema.Name, errors.New("schema for dynamic map is invalid"))
			}
			keySchema := schema.Schema[1]

			result := []map[string]any{}
			iter := objValue.MapRange()
			for iter.Next() {
				var err error
				item := map[string]any{}
				key := iter.Key()
				value := iter.Value()

				item["key"] = key.String()
				item["value"], err = normalizeToSchema(value.Interface(), keySchema)
				if err != nil {
					return nil, wrapWithSerializeError(fmt.Sprintf("%s[%q]", schema.Name, key), err)
				}

				result = append(result, item)
			}

			// Sort by key to make the output list stable, otherwise each export
			// might reorder the items making diffing harder
			sort.SliceStable(result, func(i, j int) bool {
				return strings.Compare(result[i]["key"].(string), result[j]["key"].(string)) < 0
			})
			return result, nil
		default:
			return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("schema does not match object: %d", objType.Kind()))
		}
	}

	switch schema.Type {
	case bigquery.TimestampFieldType:
		switch obj := objValue.Interface().(type) {
		case timestamppb.Timestamp:
			return time.Unix(obj.Seconds, int64(obj.Nanos)).Format(time.RFC3339), nil
		default:
			return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("convert field of type %q to timestamp", reflect.TypeOf(obj).String()))
		}
	case bigquery.StringFieldType:
		switch obj := objValue.Interface().(type) {
		case string:
			return obj, nil
		case protoreflect.Enum:
			return protoimpl.X.EnumStringOf(obj.Descriptor(), protoreflect.EnumNumber(obj.Number())), nil
		default:
			return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("convert field of type %q to string", reflect.TypeOf(obj).String()))
		}
	case bigquery.IntegerFieldType:
		switch obj := objValue.Interface().(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return obj, nil
		default:
			return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("convert field of type %q to integer", reflect.TypeOf(obj).String()))
		}
	case bigquery.FloatFieldType:
		switch obj := objValue.Interface().(type) {
		case float32, float64:
			return obj, nil
		default:
			return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("convert field of type %q to float", reflect.TypeOf(obj).String()))
		}
	case bigquery.BooleanFieldType:
		switch obj := objValue.Interface().(type) {
		case bool:
			return obj, nil
		default:
			return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("convert field of type %q to bool", reflect.TypeOf(obj).String()))
		}
	case bigquery.RecordFieldType:
		// This is a static map
		result := map[string]any{}
		for _, col := range schema.Schema {
			var err error
			// The protobuf and table fields are in snake case but the struct fields are in camel case
			fieldName := snakeCaseToCamelCase(col.Name)
			value := objValue.FieldByName(fieldName)
			if !value.IsValid() {
				// The schema might be newer than the exporter, just ignore
				continue
			}
			result[col.Name], err = normalizeToSchema(value.Interface(), col)
			if err != nil {
				return nil, wrapWithSerializeError(schema.Name, err)
			}
		}
		return result, nil
	default:
		return nil, wrapWithSerializeError(schema.Name, fmt.Errorf("unsupported bigquery field type %q", schema.Type))
	}
}

var snakeCaseWordStartRE = regexp.MustCompile(`_[\w]`)

// We keep converting the same strings in a tight loop so
// we memoize the results
var snakeCaseToCamelCaseMemoize = map[string]string{}

func snakeCaseToCamelCase(name string) string {
	result, ok := snakeCaseToCamelCaseMemoize[name]
	if !ok {
		result = strings.ToUpper(name[:1]) + snakeCaseWordStartRE.ReplaceAllStringFunc(
			name[1:],
			func(s string) string {
				return strings.ToUpper(s[1:])
			})
		snakeCaseToCamelCaseMemoize[name] = result
	}

	return result
}

type serializeError struct {
	field string
	err   error
}

func (err *serializeError) Error() string {
	return fmt.Sprintf("error serializing field %s: %v", err.field, err.err)
}

func (err *serializeError) Unwrap() error {
	return err.err
}

// wrapWithSerializeError wraps err with a serializeError. If the error is already a serializeError
// it prepends the field to the original errors field.
func wrapWithSerializeError(field string, err error) *serializeError {
	var serr *serializeError
	if errors.As(err, &serr) {
		serr.field = field + "." + serr.field
		return serr
	}

	return &serializeError{
		field: field,
		err:   err,
	}
}
