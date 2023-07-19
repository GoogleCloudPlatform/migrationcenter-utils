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

// Package messages contains user facing messages. This is done to simplify
// review of those messages.
package messages

import (
	"errors"
	"fmt"
)

// Version is the current version of the tool
var Version = "dev"

// UserAgent represents the user agent strings that will be used to identify
// the tool while accessing the varius GCP services
var UserAgent = "mc2bq/" + Version

// Message is an interface for all user facing messages
type Message interface {
	String() string
}

// SimpleMessage represents a message without parameters
type SimpleMessage string

var _ Message = SimpleMessage("")

// String implements the String method that is part of the Message interface
func (msg SimpleMessage) String() string {
	return string(msg)
}

// Exported simple messages
const (
	ExportCmdDescription SimpleMessage = `Export Migration Center data to BigQuery

    PROJECT         Project you want to export Migration Center data from. (env: MC2BQ_PROJECT)
    DATASET         Dataset that will be used to store the tables in BigQuery. If a data set with that name does not exist, one will be created. (env: MC2BQ_DATASET)
    TABLE-PREFIX    A prefix to add to the table names, this can be done to store multiple exported tables in the same data set. (env: MC2BQ_TABLE_PREFIX)`
	ParamDescriptionTargetProject SimpleMessage = "target project where the data should be exported to, if not set the project that contains the migration center data will be used. (env: MC2BQ_TARGET_PROJECT)"
	ParamDescriptionForce         SimpleMessage = "force the export of the data even if the destination table exists, the operation will delete all the content in the original table. (env: MC2BQ_FORCE)"
	ParamDescriptionSchemaPath    SimpleMessage = "use the schema at the specified path instead of using the embedded schema. (env: MC2BQ_SCHEMA_PATH)"
	ParamDescriptionRegion        SimpleMessage = "migration center region. (env: MC2BQ_REGION)"
	ParamDescriptionVersion       SimpleMessage = "print the version and exit."
	ParamDescriptionDumpSchema    SimpleMessage = "write the schema file embedded in the current version to stdout."
	ExportSuccess                 SimpleMessage = "Data exported successfully"
	ErrMsgExportTableExists       SimpleMessage = "table already exists, use --force to force the data to be overwritten"
	ErrorExportingData            SimpleMessage = "error exporting data"
	ErrorLoadingSchema            SimpleMessage = "error loading schema"
	ErrorParsingFlags             SimpleMessage = "error parsing flags"
	ErrorInvalidSchema            SimpleMessage = "invaliad schema"
)

// MissingSchemaKey represents the message that is displayed when a required
// is missing from the schema
type MissingSchemaKey struct {
	Key string
}

// String implements the String method that is part of the Message interface
func (msg MissingSchemaKey) String() string {
	return fmt.Sprintf("missing required key `%s` in schema", msg.Key)
}

// ExportCreatingDataset represents the message that is displayed when creating
// a dataset
type ExportCreatingDataset struct {
	DatasetID string
}

// String implements the String method that is part of the Message interface
func (msg ExportCreatingDataset) String() string {
	return fmt.Sprintf("Creating dataset %s...", msg.DatasetID)
}

// ExportingDataToTable represents the message that is displayed when exporting
// data to a table
type ExportingDataToTable struct {
	TableName string
}

// String implements the String method that is part of the Message interface
func (msg ExportingDataToTable) String() string {
	return fmt.Sprintf("Exporting data to table %s...", msg.TableName)
}

// NewError create an error from message
func NewError(msg Message) error {
	return errors.New(msg.String())
}

// WrapError wraps error err and prepends msg to the error string
func WrapError(msg Message, err error) error {
	return fmt.Errorf("%s: %w", msg.String(), err)
}

// ExportTableComplete is the message that is displayed when an export of a table completes
type ExportTableComplete struct {
	TableName        string
	RecordCount      uint64
	BytesTransferred uint64
}

func (msg ExportTableComplete) String() string {
	return fmt.Sprintf("Export of %s complete. %d records, %s transferred.", msg.TableName, msg.RecordCount, formatDataAmount(msg.BytesTransferred))
}

// ExportTableInProgress is the message that is displayed when an exporting to a table
type ExportTableInProgress struct {
	TableName          string
	RecordsTransferred uint64
	RecordCount        uint64
	BytesTransferred   uint64
}

func (msg ExportTableInProgress) String() string {
	if msg.RecordCount > 0 {
		return fmt.Sprintf("Export of %s in progress. %d records of %d (%d%%), %s transferred.", msg.TableName, msg.RecordsTransferred, msg.RecordCount, (msg.RecordsTransferred*100)/msg.RecordCount, formatDataAmount(msg.BytesTransferred))
	}
	return fmt.Sprintf("Export of %s in progress. %d records, %s transferred.", msg.TableName, msg.RecordsTransferred, formatDataAmount(msg.BytesTransferred))
}

// ExportComplete is the message that is displayed when the entire export completes
type ExportComplete struct {
	BytesTransferred uint64
}

func (msg ExportComplete) String() string {
	return fmt.Sprintf("Export complete. %s transferred.", formatDataAmount(msg.BytesTransferred))
}

func formatDataAmount(nBytes uint64) string {
	suffixes := []string{" bytes", "KiB", "MiB", "GiB", "TiB"}
	amount := nBytes
	for _, suffix := range suffixes {
		if amount < 1024 {
			return fmt.Sprintf("%d%s", amount, suffix)
		}
		amount /= 1024
	}

	return fmt.Sprintf("%d%s", amount, suffixes[len(suffixes)-1])
}
