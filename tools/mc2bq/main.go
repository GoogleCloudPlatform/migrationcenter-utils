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

// Package main implements the Migration Center Exporter tool
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/export"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/messages"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/schema"
)

type cliAction string

const (
	actionInvalid     = ""
	actionVersion     = "version"
	actionDumpSchema  = "dump-schema"
	actionExport      = "export"
	actionExitFailure = "exit"
)

func parseFlags(params *export.Params, argv []string) (cliAction, error) {
	var schemaPath string
	var fs flag.FlagSet

	params.DatasetID = os.Getenv("MC2BQ_DATASET")
	params.TablePrefix = os.Getenv("MC2BQ_TABLE_PREFIX")

	// set default region
	defaultRegion := "us-central1"
	if gcloudRegion := os.Getenv("REGION"); gcloudRegion != "" {
		defaultRegion = gcloudRegion
	}
	if gcloudRegion := os.Getenv("MC2BQ_REGION"); gcloudRegion != "" {
		defaultRegion = gcloudRegion
	}

	// set default project from env
	params.ProjectID = os.Getenv("PROJECT")
	if projectFromEnv := os.Getenv("MC2BQ_PROJECT"); projectFromEnv != "" {
		params.ProjectID = projectFromEnv
	}

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [FLAGS...] <PROJECT> <DATASET> [TABLE-PREFIX]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, messages.ExportCmdDescription.String())
		fmt.Fprintln(os.Stderr, "")
		fs.PrintDefaults()
	}
	fs.StringVar(
		&params.TargetProjectID,
		"target-project",
		"",
		messages.ParamDescriptionTargetProject.String())
	fs.StringVar(
		&params.Region,
		"region",
		defaultRegion,
		messages.ParamDescriptionRegion.String())
	fs.BoolVar(
		&params.Force,
		"force",
		false,
		messages.ParamDescriptionForce.String(),
	)
	fs.StringVar(
		&schemaPath,
		"schema-path",
		"",
		messages.ParamDescriptionSchemaPath.String(),
	)
	var versionFlag bool
	fs.BoolVar(&versionFlag, "version", false, messages.ParamDescriptionVersion.String())
	var dumpEmbeddedSchemaFlag bool
	fs.BoolVar(&dumpEmbeddedSchemaFlag, "dump-embedded-schema", false, messages.ParamDescriptionDumpSchema.String())
	err := fs.Parse(argv)
	if err != nil {
		return actionInvalid, err
	}
	// Keep right after parse so we ignore all other validations
	if versionFlag {
		return actionVersion, nil
	}

	// Keep right after version flag check so we ignore all other validations
	if dumpEmbeddedSchemaFlag {
		return actionDumpSchema, nil
	}

	args := fs.Args()
	if len(args) > 0 {
		params.ProjectID = fs.Arg(0)
	}
	if len(args) > 1 {
		params.DatasetID = fs.Arg(1)
	}
	if len(args) > 2 {
		params.TablePrefix = fs.Arg(2)
	}

	params.Force = params.Force || os.Getenv("MC2BQ_FORCE") != ""

	if params.ProjectID == "" || params.DatasetID == "" {
		fs.Usage()
		return actionExitFailure, nil
	}

	if params.TargetProjectID == "" {
		params.TargetProjectID = os.Getenv("MC2BQ_TARGET_PROJECT")
	}
	if params.TargetProjectID == "" {
		params.TargetProjectID = params.ProjectID
	}

	if schemaPath == "" {
		schemaPath = os.Getenv("MC2BQ_SCHEMA_PATH")
	}
	params.Schema, err = loadSchemas(schemaPath)
	if err != nil {
		return actionInvalid, err
	}

	return actionExport, nil
}

func main() {
	var params export.Params
	action, err := parseFlags(&params, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", messages.WrapError(messages.ErrorParsingFlags, err))
		os.Exit(1)
	}

	switch action {
	case actionVersion:
		fmt.Printf("mc2bq %s\n", messages.Version)
	case actionDumpSchema:
		_ = dumpEmbeddedSchema()
	case actionExport:
		err = export.Export(&params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", messages.WrapError(messages.ErrorExportingData, err))
			os.Exit(1)
		}
	case actionExitFailure:
		os.Exit(1)
	default:
		os.Exit(1)
	}

}

func loadSchemas(name string) (*schema.ExporterSchema, error) {
	if name == "" {
		// return defaults
		return &schema.EmbeddedSchema, nil
	}

	rawData, err := os.ReadFile(name)
	if err != nil {
		return nil, messages.WrapError(messages.ErrorLoadingSchema, err)
	}

	var schemas schema.ExporterSchema
	err = json.Unmarshal(rawData, &schemas)
	if err != nil {
		return nil, messages.WrapError(messages.ErrorLoadingSchema, err)
	}

	return &schemas, err
}

func dumpEmbeddedSchema() error {
	out, err := json.MarshalIndent(&schema.EmbeddedSchema, "", "  ")
	if err != nil {
		return err
	}

	os.Stdout.Write(out)
	os.Stdout.Write([]byte{'\n'})
	return nil
}
