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

// Package export contains the implmentation of the export command used to export the CAST report and Migration Center data to BigQuery.
package export

import (
	"context"
	"fmt"
	"net/http"
	"os"

	bq "cloud.google.com/go/bigquery"
	mc2bqExp "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mc2bq/pkg/export"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/root"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/gapiutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/messages"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

const (
	castTableID      = "castResults"
	defaultDatasetID = "mcCast"
)

var (
	path        string
	projectID   string
	datasetID   string
	region      string
	mcProjectID string
	mcRegion    string
	endpoint    string
	force       bool
)

type castExporterFactory interface {
	build(filePath, projectID, datasetID, tableID, location string) exporter
}

type mcExporterFactory interface {
	build(mcProjectID, mcRegion, targetProjectID, datasetID string, force bool) exporter
}

type exporter interface {
	export() error
}

// mcExporter exports data from Migration Center to BigQuery.
type mcExporter struct {
	mcProjectID     string
	mcRegion        string
	targetProjectID string
	datasetID       string
	force           bool
}

func (me *mcExporter) build(mcProjectID, mcRegion, targetProjectID, datasetID string, force bool) exporter {
	me.mcProjectID = mcProjectID
	me.mcRegion = mcRegion
	me.targetProjectID = targetProjectID
	me.datasetID = datasetID
	me.force = force
	return me
}

func (me *mcExporter) export() error {
	var opts []option.ClientOption
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	return mc2bqExp.Export(&mc2bqExp.Params{ProjectID: me.mcProjectID, Region: me.mcRegion, TargetProjectID: me.targetProjectID, DatasetID: me.datasetID, MCOptions: opts, Force: me.force})
}

// castExporter exports CAST csv files to BigQuery.
type castExporter struct {
	filePath  string
	projectID string
	datasetID string
	tableID   string
	location  string
}

func (c *castExporter) build(filePath, projectID, datasetID, tableID, location string) exporter {
	c.filePath = filePath
	c.projectID = projectID
	c.datasetID = datasetID
	c.tableID = tableID
	c.location = location
	return c
}

// export exports CAST report and Migration Center data to BigQuery.
func (c *castExporter) export() error {
	ctx := context.Background()

	// TODO varify if should be checked
	// if err := parser.ValidFileFormat(c.filePath); err != nil {
	// 	return err
	// }

	client, err := bq.NewClient(ctx, c.projectID, option.WithUserAgent(messages.ExportUserAgent))
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %w", err)
	}
	client.Location = c.location
	defer client.Close()

	f, err := os.Open(c.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	created, err := c.createDataset(ctx, client)
	if err != nil {
		return err
	}
	if created {
		fmt.Println(messages.DatasetCreated{Name: c.datasetID, Region: c.location})
	}

	if err := c.createTable(ctx, client, f); err != nil {
		return err
	}
	fmt.Println(messages.TableCreated{Name: c.tableID})

	return nil
}

// createDataset creates a data set in BigQuery. If the dataset was created returns true. If it already exists or error returns false.
func (c *castExporter) createDataset(ctx context.Context, client *bq.Client) (bool, error) {
	dataset := client.Dataset(c.datasetID)
	metadata, err := dataset.Metadata(ctx)
	if err != nil && !gapiutil.IsErrorWithCode(err, http.StatusNotFound) {
		return false, err
	}

	// Verify that the dataset exists in the requested region.
	if err == nil {
		if c.location != metadata.Location {
			return false, messages.DatasetExistError{Name: c.datasetID, CreateRegion: c.location, ExistRegion: metadata.Location}.Error()
		}
		return false, nil
	}

	// If statusNotFound == true.
	err = dataset.Create(ctx, &bq.DatasetMetadata{
		Name:     c.datasetID,
		Location: c.location,
	})
	if err != nil {
		return false, err
	}
	return true, nil

}

// createTable creates the castResults table in BigQuery and populates it with the CAST file's data.
// If the table exists and force is false, an error will be returned.
// If the table exists and force is true, the table will be rewritten.
func (c *castExporter) createTable(ctx context.Context, client *bq.Client, f *os.File) error {
	source := bq.NewReaderSource(f)
	source.AutoDetect = true   // Allow BigQuery to determine schema.
	source.SkipLeadingRows = 1 // CSV has a single header line.

	loader := client.Dataset(c.datasetID).Table(c.tableID).LoaderFrom(source)
	loader.LoadConfig.ColumnNameCharacterMap = bq.V1ColumnNameCharacterMap
	if force {
		exist, err := c.tableExist(ctx, client)
		if err != nil {
			return err
		}
		if exist {
			fmt.Println(messages.ReplacingExistingTable{Name: c.tableID})
		}
		loader.WriteDisposition = bq.WriteTruncate
	} else {
		loader.WriteDisposition = bq.WriteEmpty
	}
	job, err := loader.Run(ctx)
	if err != nil {
		return err
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}
	if err := status.Err(); err != nil {
		return err
	}
	return nil
}

func (c *castExporter) tableExist(ctx context.Context, client *bq.Client) (bool, error) {
	_, err := client.Dataset(c.datasetID).Table(c.tableID).Metadata(ctx)
	if gapiutil.IsErrorWithCode(err, http.StatusNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// newExportCmd returns the Cobra Export command representation.
// This command performs two actions:
// It creates a new table in BigQuery called 'castResults' and populates it with the CAST report data.
// It exports your Migration Center data to BigQuery.
// This results in the creation of three tables named 'assets', 'groups', and 'preference_sets' in BigQuery using data from Migration Center.
func newExportCmd(castFactory castExporterFactory, mcFactory mcExporterFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "export path project region dataset [flags]",
		Short: "Exports CAST report and Migration Center data to BigQuery.",
		Long: `Exports CAST report and Migration Center data to BigQuery.
By default it will be assumed that the project and region used for Migration Center and BigQuery are the same.`,
		Example: `
	mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 # the default dataset will be set to 'mcCast'.
	mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id 
	mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id  --force=true
	mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id --mc-project=my-mc-project-id --mc-region=my-mc-region
	`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) != 0 {
				return messages.NoArgumentsAcceptedError{Args: args}.Error()
			}

			if datasetID == "" {
				datasetID = defaultDatasetID
			}

			location := root.DefaultLocation
			if region != "" {
				location = region
			}

			// Exporting CAST file to BigQuery
			ce := castFactory.build(path, projectID, datasetID, castTableID, location)
			err := ce.export()
			if err != nil {
				if gapiutil.IsErrorWithCode(err, http.StatusConflict) {
					err = fmt.Errorf("%w,\n"+messages.ForceExport.String(), err)
				}
				return messages.WrapError(messages.ErrExportingData, err)
			}
			fmt.Println(messages.CASTExportSuccess)

			// Exporting Migration Center data to BigQurey.
			if mcProjectID == "" {
				mcProjectID = projectID
			}
			if mcRegion == "" {
				mcRegion = region
			}
			fmt.Println(messages.CallingMCToBQ{MCProjectID: mcProjectID, MCRegion: mcRegion, BQProjectID: projectID, BQRegion: region, DatasetID: datasetID})

			me := mcFactory.build(mcProjectID, mcRegion, projectID, datasetID, force)
			err = me.export()
			if err != nil {
				return messages.WrapError(messages.ErrExportingMCToBQ, err)
			}
			fmt.Println(messages.MCExportSuccess)

			fmt.Println(messages.ExportNextSteps{ProjectID: projectID, DatasetID: datasetID})
			return nil
		},
	}
}

func init() {
	exportCmd := newExportCmd(&castExporter{}, &mcExporter{})
	setExportFlags(exportCmd)
	root.RootCmd.AddCommand(exportCmd)
}

func setExportFlags(cmd *cobra.Command) {
	// Required flags.
	cmd.Flags().StringVar(&path, "path", "", `The csv file's path of the CAST report (analysisResults.csv). (required)`)
	cmd.MarkFlagRequired("path")
	cmd.Flags().StringVar(&projectID, "project", "", `The BigQuery project-id to export the data to. (required)`)
	cmd.MarkFlagRequired("project")
	cmd.Flags().StringVar(&datasetID, "dataset", "", `The dataset-id to export the data to. If the dataset doesn't exist it will be created. If not specified the default name will be 'mcCast'. Make sure to use the same dataset for every command.`)
	cmd.Flags().StringVar(&region, "region", "", `The BigQuery region in which the dataset and tables will be created. (required)`)
	cmd.MarkFlagRequired("region")

	// Optional flags.
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force the export of the data even if the destination tables exist. The operation will delete all the content in the original tables.")

	// Optional hidden flags.
	cmd.Flags().StringVar(&mcProjectID, "mc-project", "", `The Migration Center project-id from which Migration Center data will be exported to BigQuery. If not specified the default project will be the BigQuery project-id`)
	cmd.Flags().StringVar(&mcRegion, "mc-region", "", `The Migration Center region In which your data is located. This should be the region which you used for the create-groups command. If not specified this will default to your BigQuery region.`)
	cmd.Flags().MarkHidden("mc-project")
	cmd.Flags().MarkHidden("mc-region")
	cmd.Flags().StringVar(&endpoint, "mc-endpoint", "", `The endpoint Migration Centers client will use.`)
	cmd.Flags().MarkHidden("mc-endpoint")
}
