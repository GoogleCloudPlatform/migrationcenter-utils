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

// Package root contains the root command of MConnect.
package root

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	DefaultLocation = "us-central1"
	Version         = "0.1.0"
)

var (
	subcommands = []string{
		"create-groups",
		"export",
		"create-views",
	}
)

// RootCmd represents the base mconnect command when called without any subcommands.
var RootCmd = &cobra.Command{
	Use:     "mconnect [command] [args] [flags]",
	Version: Version,
	Short:   "mconnect is a tool used to export and merge information from Migration Center and CAST to BigQuery, which allows you to perform data analysis in Looker Studio.",
	Long: `mconnect is a tool used to export and merge information from Migration Center and CAST to BigQuery, which allows you to perform data analysis in Looker Studio.
Recommended Usage Steps: 
	1. Authenticate with gcloud.
	2. Run 'mconnect create-groups --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1'
	   # This command creates a group in Migration Center for each application in the CAST report file. Each group in Migration Center has the 'mconnect' label.
	3. In Migration Center, assign your assets to their corresponding application groups created in step '2'. Do this using the Migration Center UI or api.
	4. Run 'mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id'
	   # This command performs two actions:
	   # It creates a new table in BigQuery called 'castResults' and populates it with the CAST report data.
	   # It exports your Migration Center data to BigQuery. The final result in BigQuery will be the creation of 3 tables named 'assets', 'groups', and 'preference_sets' containing your data.
	5. Run 'mconnect create-views --project=my-project-id --dataset=dataset-id'.
	   # This creates three views ('migrationcenterinfra_vw', 'castreadiness_vw', 'mccastreadinesscombined_vw') in BigQuery using Migration Center and CAST data.
	   # The output of this command provides a link to a Looker Studio report using the 'mccastreadinesscombined_vw' view.
	   # Make sure to use the same BigQuery project-id and dataset-id as in step '4'.
	6. Copy the link obtained in the previous step to your web browser. Once the report loads, save it.   
	`,
	ValidArgs: subcommands,
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	RootCmd.CompletionOptions.DisableDefaultCmd = true
}
