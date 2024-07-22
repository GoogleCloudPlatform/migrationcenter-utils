# MConnect

A command line interface used to export and merge information from Migration Center and CAST to BigQuery, which allows you to perform data analysis in Looker Studio.

## Before you begin
Before you use MConnect, perform the following steps:

1. Create a Google Account and Google Cloud account.
2. Create a Google Cloud project and enable the [BigQuery](https://pantheon.corp.google.com/apis/library/bigquery.googleapis.com) and [Migration Center](https://pantheon.corp.google.com/apis/library/migrationcenter.googleapis.com) API.
    * For Migration Center, see: [Getting started with Migration Center](https://cloud.google.com/migration-center/docs/get-started-with-migration-center).
    * For BigQuery, see: [Getting started with BigQuery](https://support.google.com/cloud/answer/6255052?hl=en#zippy=%2Cget-started).
    * Migration Center currently supports regions `us-central1` and `europe-west1`.
3. Create a CAST Highlights report named 'analysisResults.csv'.
4. Populate Migration Center with the assets related to the applications in the CAST report.
    * For more information see: [Asset Discovery](https://cloud.google.com/migration-center/docs/start-asset-discovery).
5. Install the gcloud CLI.
    * For more information see: [gcloud install](https://cloud.google.com/sdk/docs/install)


## Recommended Usage Steps

### Step 1: Build from source

```sh
go build .
```

### Step 2: Authenticate with gcloud

```sh
gcloud init
gcloud auth application-default login
``` 
Make sure the account you're using has the necessary permissions to create/delete groups in Migration Center, and to create and delete tables in BigQuery in the specific project you would like to use.

### Step 3: Create groups In Migration Center

```sh
mconnect create-groups --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1
``` 

This command creates a group in Migration Center for each application in the CAST report file. Each group in Migration Center has the 'mconnect' label.

### Step 4: Assign assets to groups (**manual step**)

In Migration Center, assign your assets to their corresponding application groups created in step '3'. Do this using the [Migration Center UI](https://cloud.google.com/migration-center/docs/create-groups#add_and_remove_assets_from_a_group) or API.

### Step 5: Export CAST report and Migration Center data to BigQuery

```sh
mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id 
``` 

This command performs two actions:

1. It creates a new table in BigQuery called 'castResults' and populates it with the CAST report data.
2. It exports your Migration Center data to BigQuery. The final result in BigQuery will be the creation of a dataset which has three tables named 'assets', 'groups', and 'preference_sets' containing your data.

### Step 6: Create views in BigQuery

```sh
mconnect create-views --project=my-project-id --dataset=dataset-id
``` 
This creates three views ('migrationcenterinfra_vw', 'castreadiness_vw', 'mccastreadinesscombined_vw') in BigQuery using Migration Center and CAST data.
The output of this command provides a link to a Looker Studio report using the 'mccastreadinesscombined_vw' view.

Make sure to use the **same** project-id and dataset-id as in step 5.

### Step 7: Setup Looker Studio's Report

#### Using the provided link

1. Copy the link obtained in the previous step to your web browser.
2. Click 'Save and Share' and then 'Acknowledge and save'.

#### Manually copying -

If the link provided is broken, you could manually set up the Looker Studio Report using your data:

1. In Looker Studio, open the 'Migration Center / CAST Analysis' report, click the three dots at the top of the page.
2. Click 'Make a copy', and then 'Copy Report'.
3. In the new report, click 'resources', and then 'Manage added data sources'.
4. Using the datasource named: 'McCastReadinessCombined_vw', click 'EDIT'.
5. Provide the project-id and the dataset-id used in step '6' and choose 'mccastreadinesscombined_vw'.
6. Click 'RECONNECT', and then 'Apply'.
7. Click 'DONE' and refresh the page.

This creates a 'Migration Center / CAST Analysis' report using your data.

## Usage

### mconnect

```text 
Usage: mconnect [command] [args] [flags]

Available Commands:
  create-groups Creates a group for each Cast application in MC and adds a 'mconnect' label to it.
  create-views  Creates three views in BigQuery Using Migration Center and CAST's data.
  export        Exports Cast’s data to BigQuery.
  help          Help about any command

Flags:
  -h, --help     Help for mconnect
  -t, --toggle   Help message for toggle
  -v, --version  Version for mconnect

```

### create-groups

```text
Creates a group for each CAST application in Migration Center and adds the 'mconnect' label to it.

Usage:
  mconnect create-groups path project region [flags]

Examples:

                mconnect create-groups --path=path/to/cast/analysisResults.csv --project=my-mc-project-id --region=my-region1
                mconnect create-groups --path=path/to/cast/analysisResults.csv --project=my-mc-project-id --region=my-region1 --ignore-existing-groups=true

Flags:
  -h, --help                     Help for create-groups
  -i, --ignore-existing-groups   Continue if mconnect is trying to create a group that already exists in Migration Center.
                                 If set to 'true', the 'mconnect' label will be added to every group that already exists as well.
      --path string              The csv file's path which contains CAST's report (analysisResults.csv). (required)
      --project string           The project-id in which to create the Migration Center groups. Make sure to use the same Project ID for every command. (required)
      --region string            The Migration Center region in which the groups will be created. (required)

```

### export

```text
Exports CAST report and Migration Center data to BigQuery.
By default it will be assumed that the project and region used for Migration Center and BigQuery are the same.

Usage:
  mconnect export path project region dataset [flags]

Examples:

        mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 # the default dataset will be set to 'mcCast'.
        mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id 
        mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id  --force=true
        mconnect export --path=path/to/cast/analysisResults.csv --project=my-project-id --region=my-region1 --dataset=dataset-id --mc-project=my-mc-project-id --mc-region=my-mc-region

Flags:
      --dataset string   The dataset-id to export the data to. If the dataset doesn't exist it will be created. If not specified the default name will be 'mcCast'. Make sure to use the same dataset for every command.
  -f, --force            Force the export of the data even if the destination tables exist. The operation will delete all the content in the original tables.
  -h, --help             Help for export
      --path string      The csv file's path of the CAST report (analysisResults.csv). (required)
      --project string   The BigQuery project-id to export the data to. (required)
      --region string    The BigQuery region in which the dataset and tables will be created. (required)

Hidden Flags:
      --mc-project string	The Migration Center project-id used to export its data to BigQuery.
      --mc-region string	The Migration Center region from which to export the data.

```



### create-views

```text
Creates three views in BigQuery using Migration Center and CAST data.
Provides a link for a Looker Studio report using the 'mccastreadinesscombined_vw' view.

Views created:
        migrationcenterinfra_vw - Shows grouped asset data from Migration Center.
        castreadiness_vw - Shows data from the CAST Analysis file.
        mccastreadinesscombined_vw - Combines the two previous views. This view is also used in Looker Studio's Template.

Usage:
  mconnect create-views project dataset [flags]

Examples:

mconnect create-views --project=my-project-id --dataset=dataset-id
mconnect create-views --project=my-project-id --dataset=dataset-id --force=true

Flags:
      --dataset string   The BigQuery dataset-id to create the views in. Make sure to use the same dataset as in the export command. (required)
  -f, --force            Force the creation of views even if only one of the destination views exist. The operation will replace all the contents in the old existing views.
  -h, --help             Help for create-views
      --project string   The BigQuery project-id to create the views in. (required)

```




