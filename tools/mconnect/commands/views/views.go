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

// Package views contains the implmentation of the create-views command used to create views in BigQuery using CAST and Migration Center data and generate a link
// to a looker studio report using that data.
package views

import (
	"context"
	"fmt"
	"net/http"

	bq "cloud.google.com/go/bigquery"
	gp "github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/groups"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/commands/root"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/gapiutil"
	"github.com/GoogleCloudPlatform/migrationcenter-utils/tools/mconnect/messages"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
)

const (
	mcViewName       = "migrationcenterinfra_vw"
	castViewName     = "castreadiness_vw"
	combinedViewName = "mccastreadinesscombined_vw"

	lookerReportID = "f05dec2f-fa92-4b8b-b379-a067bfdd8b09"
)

var (
	projectID string
	datasetID string
	force     bool
)

type viewMetadata struct {
	name        string
	description string
	query       string
}

func newViewMetadata(name, description, query string) viewMetadata {
	return viewMetadata{
		name:        name,
		description: description,
		query:       query,
	}
}

type viewCreatorFactory interface {
	build(projectID, datasetID string) viewCreator
}

type viewCreator interface {
	createView(ctx context.Context, metadata viewMetadata) error
}

type bqViewCreator struct {
	projectID string
	datasetID string
}

func (vc *bqViewCreator) build(projectID, datasetID string) viewCreator {
	vc.projectID = projectID
	vc.datasetID = datasetID
	return vc
}

// createView creates a view in BigQuery specified by the view metadata.
// If the view already exists and force is false an error will be returned.
// If the view already exists and force is true, the old view will be overwritten.
func (vc *bqViewCreator) createView(ctx context.Context, metadata viewMetadata) error {
	client, err := bq.NewClient(ctx, vc.projectID, option.WithUserAgent(messages.ViewsUserAgent))
	if err != nil {
		return fmt.Errorf("bigquery.NewClient: %v", err)
	}

	defer client.Close()

	bqMetaData := &bq.TableMetadata{Name: metadata.name, Description: metadata.description, ViewQuery: metadata.query}
	if force {
		oldMetadata, err := client.Dataset(vc.datasetID).Table(metadata.name).Metadata(ctx)
		if err != nil && !gapiutil.IsErrorWithCode(err, http.StatusNotFound) {
			return err
		}
		if oldMetadata != nil {
			err := client.Dataset(vc.datasetID).Table(metadata.name).Delete(ctx)
			if err != nil {
				return err
			}
			fmt.Println(messages.ViewDeleted{Name: oldMetadata.Name})
		}
	}

	if err := client.Dataset(vc.datasetID).Table(metadata.name).Create(ctx, bqMetaData); err != nil {
		return err
	}
	fmt.Println(messages.ViewCreated{Name: metadata.name})
	return nil
}

// NewcreateViewsCmd returns the createViews command.
func NewCreateViewsCmd(factory viewCreatorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "create-views project dataset",
		Short: "Creates three views in BigQuery using Migration Center and CAST data and outputs a link to a Looker Studio report using these views.",
		Long: `Creates three views in BigQuery using Migration Center and CAST data.
Provides a link for a Looker Studio report using the 'mccastreadinesscombined_vw' view.

Views created:
	migrationcenterinfra_vw - Shows grouped asset data from Migration Center.
	castreadiness_vw - Shows data from the CAST Analysis file.
	mccastreadinesscombined_vw - Combines the two previous views. This view is also used in Looker Studio's Template.


`,
		Example: `
mconnect create-views --project=my-project-id --dataset=dataset-id	
mconnect create-views --project=my-project-id --dataset=dataset-id --force=true`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) != 0 {
				return messages.NoArgumentsAcceptedError{Args: args}.Error()
			}

			vc := factory.build(projectID, datasetID)
			viewsMetadata := []viewMetadata{
				newViewMetadata(mcViewName, "mc view description", mcQuery(projectID, datasetID)),
				newViewMetadata(castViewName, "cast view description", castQuery(projectID, datasetID)),
				newViewMetadata(combinedViewName, "combined view description", combinedQuery(projectID, datasetID)),
			}
			for _, metadata := range viewsMetadata {
				err := vc.createView(ctx, metadata)
				if err != nil {
					if gapiutil.IsErrorWithCode(err, http.StatusConflict) {
						return fmt.Errorf(messages.CreatingViewExistError{Name: metadata.name}.String())
					}

					return messages.CreatingViewError{Metadata: metadata.name, Err: err}.Error()
				}
			}

			// Prints the Looker Studio Link with the users data connected to it.
			fmt.Println(messages.LookerLinkInstruction{Link: lookerStudioLink(projectID, datasetID)})

			return nil
		},
	}
}

func init() {
	createViewsCmd := NewCreateViewsCmd(&bqViewCreator{})
	setCreateViewsFlags(createViewsCmd)
	root.RootCmd.AddCommand(createViewsCmd)
}

func setCreateViewsFlags(cmd *cobra.Command) {
	// Required flags.
	cmd.Flags().StringVar(&projectID, "project", "", `The BigQuery project-id to create the views in. (required)`)
	cmd.MarkFlagRequired("project")
	cmd.Flags().StringVar(&datasetID, "dataset", "", `The BigQuery dataset-id to create the views in. Make sure to use the same dataset as in the export command. (required)`)
	cmd.MarkFlagRequired("dataset")

	// Optional flags.
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force the creation of views even if only one of the destination views exist. The operation will replace all the contents in the old existing views.")
}

func mcQuery(projectID, datasetID string) string {
	// The query is built with fmt.Printf since creating views can't do any update/delete operations, hence won't be able to ruin any data.
	// Also, as this is intended to be used locally, it is safe to assume a user wouldn't try to inject parameters that would mess his project.
	assetsTable := fmt.Sprintf("%v.%v.assets", projectID, datasetID)
	groupsTable := fmt.Sprintf("%v.%v.groups", projectID, datasetID)

	query := `WITH FilteredGroups AS (
		SELECT name, display_name
		FROM %v
		WHERE EXISTS (
		  SELECT 1
		  FROM UNNEST(labels) AS label
		  WHERE label.key = '%v'
		)
	  )
	  
	  SELECT grp.display_name as Application, machine_details.machine_name AS VMs, machine_details.core_count AS vCPUs, machine_details.memory_mb/1024 AS MemoryGBs, machine_details.disks.total_capacity_bytes/(1024*1024*1024) AS StorageGBs
	  FROM %v AS assets
	  CROSS JOIN
		UNNEST(assets.assigned_groups) AS application
	  INNER JOIN
		FilteredGroups AS grp
	  ON
		application = grp.name`

	return fmt.Sprintf(query, groupsTable, gp.LabelKey, assetsTable)
}

func castQuery(projectID, datasetID string) string {
	castTable := fmt.Sprintf("%v.%v.castResults", projectID, datasetID)
	return fmt.Sprintf("SELECT  Application,  Business_Units AS BusinessUnits,  BusinessValue,  CloudReady AS CloudReadyScore,  REPLACE(Technologies, \";\", \"\\n\") AS Technologies,  Software_Resiliency,  Roadblocks,  Lines_of_Code,  Digital_Readiness AS DigitalReadiness,  Technical_Debt__min__/10080 AS TechnicalDebtWeeks FROM   `%v` WHERE Business_Units IS NOT NULL", castTable)

}
func combinedQuery(projectID, datasetID string) string {
	mcViewPath := fmt.Sprintf("%v.%v.%v", projectID, datasetID, mcViewName)
	castViewPath := fmt.Sprintf("%v.%v.%v", projectID, datasetID, castViewName)
	return fmt.Sprintf("SELECT CastR.Application, REPLACE(CastR.BusinessUnits, \";\", \"\\n\") as BusinessUnits, AVG(CastR.BusinessValue) as BusinessValue,  COUNT(McInfra.VMs) AS VMs,  SUM(McInfra.vCPUs) AS vCPUs, SUM(McInfra.MemoryGBs) AS MemoryGBs, SUM(McInfra.StorageGBs) AS StorageGBs, AVG(CastR.CloudReadyScore) AS CloudReadyScore,  CastR.Technologies, AVG(CastR.Software_Resiliency) as SoftwareResiliency,  CastR.Roadblocks, AVG(CastR.Lines_of_Code) AS LinesOfCode,  AVG(CastR.DigitalReadiness) AS DigitalReadiness,  AVG(CastR.TechnicalDebtWeeks) AS TechnicalDebtWeeks FROM  `%v` AS McInfra INNER JOIN `%v` AS CastR ON McInfra.Application = CastR.Application GROUP BY  CastR.Application, CastR.BusinessUnits,  CastR.Roadblocks,  CastR.Technologies", mcViewPath, castViewPath)
}

func lookerStudioLink(projectID, datasetID string) string {
	return fmt.Sprintf(`https://lookerstudio.google.com/c/u/0/reporting/create?c.reportId=%v&c.mode=edit&ds.ds5.datasourceName=MConnect&ds.ds5.connector=bigQuery&ds.ds5.projectId=%v&ds.ds5.type=CUSTOM_QUERY&ds.ds5.sql=SELECT%%20*%%20FROM%%20%%20%v.%v.%v`, lookerReportID, projectID, projectID, datasetID, combinedViewName)
}
