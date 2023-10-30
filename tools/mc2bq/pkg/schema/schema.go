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
func NewSerializer[T protoreflect.ProtoMessage](root string, schema bigquery.Schema) func(obj T) ([]byte, error) {
	return func(obj T) ([]byte, error) {
		return SerializeObjectToBigQuery(obj.ProtoReflect(), root, schema)
	}
}

// SerializeObjectToBigQuery serializes an object as a BigQuery compatible JSON.
// A '\n' is appended at the end of the json data.
// The function should never return an error in production, if it fails it's a bug
// resulting from a mismatch between the API object and the BigQuery schema and both are generated
// from the same protobuf.
func SerializeObjectToBigQuery(obj protoreflect.Message, root string, schema bigquery.Schema) ([]byte, error) {
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

func fieldConversionError(kind protoreflect.Kind, bqtype bigquery.FieldType) error {
	return fmt.Errorf("convert proto kind %q to bigquery type %q", kind.String(), bqtype)
}

func convertProtoValueToBQType(value protoreflect.Value, fd protoreflect.FieldDescriptor, schema *bigquery.FieldSchema) (any, error) {
	bqtype := schema.Type
	kind := fd.Kind()
	switch kind {
	case protoreflect.BoolKind:
		switch bqtype {
		case bigquery.BooleanFieldType:
			return value.Bool(), nil
		}
	case protoreflect.Int32Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind,
		protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind:
		switch bqtype {
		case bigquery.IntegerFieldType:
			return value.Int(), nil
		}
	case protoreflect.Uint32Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind:
		switch bqtype {
		case bigquery.IntegerFieldType:
			return value.Uint(), nil
		}
	case protoreflect.StringKind:
		switch bqtype {
		case bigquery.StringFieldType:
			return value.String(), nil
		}
	case protoreflect.FloatKind,
		protoreflect.DoubleKind:
		return value.Float(), nil
	case protoreflect.MessageKind:
		return normalizeToSchema(value.Message(), schema)
	case protoreflect.EnumKind:
		switch bqtype {
		case bigquery.StringFieldType:
			return protoimpl.X.EnumStringOf(fd.Enum(), value.Enum()), nil
		}
	}
	return nil, fieldConversionError(kind, bqtype)
}

func normalizeMessageField(obj protoreflect.Message, col *bigquery.FieldSchema) (any, error) {
	fieldDesc := obj.Descriptor().Fields().ByName(protoreflect.Name(col.Name))
	if fieldDesc == nil {
		return nil, fmt.Errorf("field %q not found", col.Name)
	}
	value := obj.Get(fieldDesc)

	if fieldDesc.Cardinality() != protoreflect.Repeated {
		res, err := convertProtoValueToBQType(value, fieldDesc, col)
		if err != nil {
			return nil, wrapWithSerializeError(col.Name, err)
		}

		return res, nil
	}

	if fieldDesc.IsList() {
		lst := value.List()
		if lst.Len() == 0 {
			return nil, nil
		}

		tmpList := make([]any, lst.Len())
		itemSchema := *col
		itemSchema.Repeated = false // we are serializing the item so it's not repeated
		for i := 0; i < len(tmpList); i++ {
			var err error
			item := lst.Get(i)
			tmpList[i], err = convertProtoValueToBQType(item, fieldDesc, &itemSchema)
			if err != nil {
				return nil, wrapWithSerializeError(fmt.Sprintf("%s[%d]", col.Name, i), err)
			}
		}

		return tmpList, nil
	}

	if fieldDesc.IsMap() {
		// Create a dynamic map. Because maps are not supported by BigQuery.
		// We need to convert it to an array in the form of [struct{key: string, value: T}, ...]

		if len(col.Schema) != 2 {
			return nil, wrapWithSerializeError(col.Name, errors.New("schema for dynamic map is invalid"))
		}

		if col.Schema[1].Name != "value" {
			return nil, wrapWithSerializeError(col.Name, errors.New("schema for dynamic map is invalid"))
		}

		if fieldDesc.MapKey().Kind() != protoreflect.StringKind {
			return nil, wrapWithSerializeError(col.Name, errors.New("schema for dynamic map is invalid"))
		}

		dict := value.Map()
		keySchema := col.Schema[1]
		result := []map[string]any{}

		var err error
		dict.Range(func(mk protoreflect.MapKey, v protoreflect.Value) bool {
			item := map[string]any{}
			key := mk.String()
			item["key"] = key
			item["value"], err = convertProtoValueToBQType(v, fieldDesc.MapValue(), keySchema)
			if err != nil {
				err = wrapWithSerializeError(fmt.Sprintf("%s[%q]", col.Name, key), err)
				return false
			}

			result = append(result, item)
			return true
		})

		if err != nil {
			return nil, err
		}

		// Sort by key to make the output list stable, otherwise each export
		// might reorder the items making diffing harder
		sort.SliceStable(result, func(i, j int) bool {
			return strings.Compare(result[i]["key"].(string), result[j]["key"].(string)) < 0
		})

		if len(result) == 0 {
			return nil, nil
		}

		return result, nil
	}

	// Fields are either scalars, lists or maps
	panic("unreachable code")
}

// normalizeToSchema normalizes obj to according to the provided schema.
// Specifically converts structs to maps, and maps to key-value lists.
func normalizeToSchema(obj protoreflect.Message, schema *bigquery.FieldSchema) (any, error) {
	// short circuit for nil values
	if !obj.IsValid() {
		return nil, nil
	}

	if obj, ok := obj.Interface().(*timestamppb.Timestamp); ok {
		return time.Unix(obj.Seconds, int64(obj.Nanos)).Format(time.RFC3339), nil
	}

	result := map[string]any{}

	for _, col := range schema.Schema {
		res, err := normalizeMessageField(obj, col)
		if err != nil {
			return nil, err
		}

		if res == nil {
			continue
		}

		result[col.Name] = res
	}

	return result, nil
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
