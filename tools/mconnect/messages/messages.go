/*
Copyright 2024 Google LLC All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package messages contains the messages printed out to the user.
package messages

import (
	"fmt"
	"path"
)

// Version is the current version of the tool.
var Version = "dev"

// UserAgent represents the user agent strings that will be used to identify
// the tool while accessing the varius GCP services.
var GroupsUserAgent = "mconnect/create-groups/" + Version
var ExportUserAgent = "mconnect/export/" + Version
var ViewsUserAgent = "mconnect/create-views/" + Version

// Message is an interface for all user facing messages.
type Message interface {
	String() string
}

// SimpleMessage represents a message without parameters.
type SimpleMessage string

var _ Message = SimpleMessage("")

// String implements the String method that is part of the Message interface.
func (msg SimpleMessage) String() string {
	return string(msg)
}

// Exported simple messages.
const (
	GroupsCreation    SimpleMessage = "Creating groups..."
	GroupsSuccess     SimpleMessage = "Groups created successfully"
	CASTExportSuccess SimpleMessage = "CAST data exported successfully..."
	MCExportSuccess   SimpleMessage = "Migration Center data exported successfully..."
	ForceExport       SimpleMessage = "to force the export use the 'force' flag. Doing so will delete all the content in the original table"

	ErrMissingPath    SimpleMessage = "missing required 'path' flag."
	ErrMissingProject SimpleMessage = "missing required 'project-id' flag."

	ErrMsgExportTableExists SimpleMessage = "table already exists, use '--force=true' to force the data to be overwritten"
	ErrExportingData        SimpleMessage = "failed to export CAST data"
	ErrExportingMCToBQ      SimpleMessage = "failed to export data from Migration Center to BigQuery"
)

type ParsingFile struct {
	FilePath string
}

func (msg ParsingFile) String() string {
	return fmt.Sprintf("Parsing file '%v'...", path.Base(msg.FilePath))
}

type WrongFileFormatError struct {
	FileFormat string
}

func (msg WrongFileFormatError) Error() error {
	return fmt.Errorf("wrong CAST file format, want: '.csv' or '.txt', got: '%v'", msg.FileFormat)
}

type GroupsApps struct {
	Applications int
}

func (msg GroupsApps) String() string {
	return fmt.Sprintf("Found %d applications...", msg.Applications)
}

type GroupCreated struct {
	Group string
}

func (msg GroupCreated) String() string {
	return fmt.Sprintf("Group '%v' was created successfully...", msg.Group)
}

type GroupExists struct {
	Group string
}

func (msg GroupExists) String() string {
	return fmt.Sprintf("Group '%v' already exists...", msg.Group)
}

type GroupDetectedLabel struct {
	Group string
	Label string
}

func (msg GroupDetectedLabel) String() string {
	return fmt.Sprintf("Group '%v' has label '%v', no update necessary...", msg.Group, msg.Label)
}

type GroupUpdated struct {
	Name string
}

func (msg GroupUpdated) String() string {
	return fmt.Sprintf("Group '%v' updated...", msg.Name)
}

type GroupsNextSteps struct {
	Path      string
	ProjectID string
	Region    string
}

func (msg GroupsNextSteps) String() string {
	return fmt.Sprintf(`Recommended next steps:
	1. In Migration Center, assign your assets to their corresponding application groups that were just created. Do this using the Migration Center UI or API. 
	2. Run: 'mconnect export --path=%v --project=%v --region=%v'`, msg.Path, msg.ProjectID, msg.Region)
}

type DatasetCreated struct {
	Name   string
	Region string
}

func (msg DatasetCreated) String() string {
	return fmt.Sprintf("Dataset '%v' was created in region: '%v'...", msg.Name, msg.Region)
}

type DatasetExistError struct {
	Name         string
	CreateRegion string
	ExistRegion  string
}

func (msg DatasetExistError) Error() error {
	return fmt.Errorf("dataset '%v' couldn't be created in region '%v' because it already exists in region '%v'", msg.Name, msg.CreateRegion, msg.ExistRegion)
}

type ReplacingExistingTable struct {
	Name string
}

func (msg ReplacingExistingTable) String() string {
	return fmt.Sprintf("Replacing existing '%v' table...", msg.Name)
}

type TableCreated struct {
	Name string
}

func (msg TableCreated) String() string {
	return fmt.Sprintf("Table '%v' was created successfully...", msg.Name)
}

type CallingMCToBQ struct {
	MCProjectID string
	MCRegion    string
	BQProjectID string
	BQRegion    string
	DatasetID   string
}

func (msg CallingMCToBQ) String() string {
	return fmt.Sprintf(`Exporting Migration Center data from project '%v' in region '%v' to BigQuery project '%v', dataset '%v' in region '%v'`, msg.MCProjectID, msg.MCRegion, msg.BQProjectID, msg.DatasetID, msg.BQRegion)
}

type ExportNextSteps struct {
	ProjectID string
	DatasetID string
}

func (msg ExportNextSteps) String() string {
	return fmt.Sprintf(`Recommended next steps: 
	1. Run: 'mconnect create-views --project=%v --dataset=%v'`, msg.ProjectID, msg.DatasetID)
}

type ViewDeleted struct {
	Name string
}

func (msg ViewDeleted) String() string {
	return fmt.Sprintf("Old '%v' view was deleted...", msg.Name)
}

type ViewCreated struct {
	Name string
}

func (msg ViewCreated) String() string {
	return fmt.Sprintf("View '%v' was created successfully...", msg.Name)
}

type CreatingViewExistError struct {
	Name string
}

func (msg CreatingViewExistError) String() string {
	return fmt.Sprintf("view '%v' already exists. To force the creation use the 'force' flag. Doing so will replace the old view.", msg.Name)
}

type CreatingViewError struct {
	Metadata string
	Err      error
}

func (msg CreatingViewError) Error() error {
	return fmt.Errorf("failed creating view: %v, err: %v", msg.Metadata, msg.Err)
}

type LookerLinkInstruction struct {
	Link string
}

type NoArgumentsAcceptedError struct {
	Args []string
}

func (msg NoArgumentsAcceptedError) Error() error {
	return fmt.Errorf("command accepts only flags but got arguments: %v", msg.Args)
}

func (msg LookerLinkInstruction) String() string {
	return fmt.Sprintf("Follow this link to view you data in Looker Studio: %v", msg.Link)
}

// WrapError wraps error err and prepends msg to the error string
func WrapError(msg Message, err error) error {
	return fmt.Errorf("%s, err: %w", msg.String(), err)
}
