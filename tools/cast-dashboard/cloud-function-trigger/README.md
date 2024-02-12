# README.md for Terraform Configuration

## Overview

This Terraform configuration is designed to set up a cloud environment in Google Cloud Platform (GCP) for handling data importing and migration assessment functions. The configuration automates the process of enabling necessary GCP services, setting up Cloud Functions, and managing required permissions.



## Setup Instructions

1. **Clone the Repository:**
   - Clone or download this repository to your local machine.

2. **Google Cloud Storage (GCS) Bucket:**
   - Create a GCS bucket in your Google Cloud project or use an existing one.
   - Go to the folder [data_importer_function](https://github.com/bishtkomal/mc-cast-tf/tree/main/cloud-function-trigger/data_importer_function) and download the code as zip. Repeat the same with the folder [migration_assessment_function](https://github.com/bishtkomal/mc-cast-tf/tree/main/cloud-function-trigger/migration_assessment_function).
   - Upload the `data_importer_source.zip` and `migration_assessment_source.zip` files to the bucket created in step 1.
   - **Note:** Keep the names of the zip same as mentioned i.e. **`data_importer_source.zip`** and **`migration_assessment_source.zip`** respectively.

3. **Configuration Files:**
   - `main.tf`: Contains the main Terraform configuration for setting up GCP services.
   - `variables.tf`: Define your project-specific variables here.
   - `outputs.tf`: Contains outputs that will be shown post Terraform execution.
   - `terraform.tfvars`: Update this file with your specific values for project ID, region, bucket name, dataset, and table IDs.

4.  **Update `terraform.tfvars` file:**
   
|  Variable name | Description   | Sample values  |
| ------------- |:-------------:| -----:|
| project_id      | Project ID of the project where resources need to be deployed. ( Same as project of Migration Center ) | "test-vz-2" |
| gcp_region      | Region where your resources like cloud functions will be created. NOTE: Cloud function is a regional service. Hence please provide a regional value | 'us-central1' |
| bucket_name |  Bucket where .zip and Cast files will be uploaded     |  "mc-cast-bckt"   |
| bigquery_dataset      | Dataset Id for the dataset where Cast tables will be created.      |   "cast_data" |
| bigquery_dataset_location |  This is the location where your Bigquery Cast dataset will be created. Please keep in mind that this location should be same as the location of the dataset where migration center tables: assets, groups & preference_sets lie.     |    “US” |
| bigquery_table      | Table name for cast data      |   "cast_analysis_results" |
| migrationcenter_path | This value is to be specified for the cloud function that will create groups in the migration center. | This value is of the pattern “projects/{project_id}/locations/{region of mc}”. Example: “projects/test-vz-2/locations/us-central1” |


5.  **Initialize Terraform:**
   - Open a terminal and navigate to the directory containing your Terraform files.
   - Run `terraform init`. This command initializes Terraform and downloads the necessary providers.

6. **Apply Terraform Configuration:**
   - Run `terraform apply`.
   - Review the plan and type `yes` to proceed with the configuration.

7. **Verify:**
   - Once Terraform has successfully applied the configuration, verify in your GCP console that all services are enabled and the resources are created.
  
8. **Upload Cast File & trigger the cloud functions**
   - After validating all the resources, now its time to execute this pipeline.
   - Upload your cast file namely: "analysisResults.txt" to the bubket created in steps 1.
   - As soon as the file has been uploaded to the bucket, both the cloud functions should be triggered.
   - Execution of the cloud function : data_importer_function will create a table loaded with data from the file uploaded in the bucket
   - Execution of the cloud function : migration_assessment_function will create a goups in migration center
  
9. **Map assets to groups**
    - After creation of groups is successfull, we now need to map assets to these groups.
    - As of now this is a manual process and needs to be handled through the UI.

## Expected Results from terraform

- This setup will enable necessary APIs like Cloud Functions, Artifact Registry, and others in your GCP project.
- Pub/Sub topics for these functions will also be set up i.e `import-topic` and `assessment-topic`, and necessary IAM bindings will be configured.

- Two Cloud Functions will be deployed: `data_importer_function` and `migration_assessment_function`.
1) `data_importer_function`  - Will import csv data into BigQuery tables
2) `migration_assessment_function` - Will be responsible for creating groups in migration center.
Expectation : These functions should be triggered as soon as cast files land into the GCS bucket.

- BigQuery Tables
1) Cast Analysis Results - <project_id>.<dataset_id>.<table_name>


