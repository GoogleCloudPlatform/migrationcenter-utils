# Architecture 
![alt text](https://github.com/varunika/migrationcenter-utils/blob/experimental/tools/cast-dashboard/Architecture%20Diagram.png)

# Overall Data Flow

> [!Note]
> The green bullets from **${\color{green}1.}$** to **${\color{green}5.}$** represent the ${\color{green}green \space bubbles}$ ( 1 to 5 ) in the architecture diagram, in the order of execution.

**${\color{green}1.}$** User Clones github repo to local.


**${\color{green}2.}$** User sets up the [MC2BQ tool](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/mc2bq) by following the instructions in the README file.
   * Successful set up of MC2BQ tool should result in creation of a Cloud Run job that can either be run manually on adhoc basis or scheduled to run periodically.
   * The User runs the job to populate the raw tables in Bigquery.
   * Once all the 3 tables: assets, groups and preference_sets have been populated, next steps can be followed to set up the mc-cast dashboard utility.

**${\color{green}3.}$** The User scans their application using the CAST highlights tool.
   * The Cast highlights tool generates txt files as output viz. surveyResults.txt and analysisResults.txt

**${\color{green}4.}$** The MC-cast dashboard utility has two portions - 
   * [bigquery-views](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/cast-dashboard/bigquery-views)
   * [cloud-function-trigger](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/cast-dashboard/cloud-function-trigger)
   
   The user will first set up only the [cloud-function-trigger](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/cast-dashboard/cloud-function-trigger) part of the utility.
   
**${\color{green}5.}$** The User uploads analysisResults.txt to Google cloud storage. (Manual process)
   * Two event driven functions i.e Data Loader and Mc Group function are triggered as soon as a file lands in the GCS bucket.
   * The Data Loader function populates the cast file data into Bigquery raw table
   * The Mc group function reads the cast file to find the list of applications. These applications are created as groups in the Migration Center console.

6. Once both cloud functions have run successfully, the user needs to map the groups to the assets in the Migration Center.
7. After mapping all assets to groups successfully, re-run the [CloudRun job](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/mc2bq) to populate the groups table in BigQuery.

8. The user can now set up the [bigquery-views](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/cast-dashboard/bigquery-views) part of the cast-dashboard utility by following the instructions in the README file.
9. Successful execution of step 8 will result in the final view - **[McCastReadinessCombined_vw](https://github.com/varunika/migrationcenter-utils/tree/experimental/tools/cast-dashboard/bigquery-views#2-views)** which will be used in the Looker studio dashboard.

10. Open the [template link](https://lookerstudio.google.com/c/reporting/f05dec2f-fa92-4b8b-b379-a067bfdd8b09/page/p_hcrd9nhkbd/preview). Click on “use my own data”
11. Select the project id and dataset where McCastReadinessCombined_vw was created.
12. Select McCastReadinessCombined_vw as the data source and click on done.


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

#### RESULT

1. A new dataset is created namely “migration_center” (default name)
2. Three Tables are created in the dataset:
   * Assets - Contains information about the workloads ( VMs) and mapping to groups.
   * Groups - Contains information about application groups - their Ids, names, descriptions, labels if any.
   * Preference Sets

> [!NOTE]
> As of now, the below steps require default variable values to be used in mc2bq tool.

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
