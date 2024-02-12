# Components


As of now, there are 4 different components that are required to be setup by the user inorder to create the dashboards

1) GCP Cloud run job that extracts data from Migration Centre to BigQuery.
2) 2 Cloud Functions. One that loads cast files to Bigquery and the other that triggers creation groups in Migration Center.
3) BigQuery Views for Looker Studio via Terraform
4) Looker Studio Dashboard


# Setup

### 1. Cloud Run Job (MC2BQ tool)
The Migration Center to BigQuery is an open source utility which helps you export information from Google Cloud Migration Center to BigQuery tables. The tool runs as a cloud run job. The following instructions will create the required job for exporting the information required from the migration center.  
To setup the MC2BQ cloud run job please follow the instructions listed [here](https://github.com/GoogleCloudPlatform/migrationcenter-utils/tree/main/tools/mc2bq#readme)

*NOTE : As of now, the below steps require default variable values to be used in mc2bq tool.

### 2. Cloud Functions
Please open the [Cloud Function Trigger folder](https://github.com/bishtkomal/mc-cast-tf/tree/main/cloud-function-trigger) and follow instructions listed in the ReadMe.

### 3. BigQuery Views
Please open [the BigQuery Views folder](https://github.com/bishtkomal/mc-cast-tf/tree/main/bigquery-views) and follow instructions listed in the ReadMe.

### 4. Looker Studio Dashboard
Assumptions:
1) All set up steps from 1 to 3 mentioned above have been completed.
2) Cast files have been loaded to BigQuery.
3) Migration Center Cloud run job has also run atleast once.

The Looker studio dashboard can be created by using [The MC / CAST Dashboard Template](https://lookerstudio.google.com/c/reporting/f05dec2f-fa92-4b8b-b379-a067bfdd8b09/page/p_hcrd9nhkbd/preview).

Just click on "Use my own data" button and change the data source to the new view in BigQuery created by step 3.
