# README.md for Terraform BigQuery Configuration

## Introduction

This Terraform configuration is designed to set up a BigQuery dataset and a set of tables (or views) in Google Cloud Platform (GCP). It dynamically creates BigQuery tables based on provided configurations, making it an efficient tool for setting up complex data environments in GCP.

## Assumptions
1) The User has already set up and run Cloud run job for Migration Center to BigQuery
1) Raw tables containing data from the migration center already exist in BigQuery.
1) Terraform is already installed. If not installed already, follow the steps listed [here](https://developer.hashicorp.com/terraform/tutorials/gcp-get-started/install-cli).

## Configuration Details

- `main.tf`: Contains the Terraform resources required to create a BigQuery dataset and tables.
- `variables.tf`: Declares variables used in `main.tf`.
- `terraform.tfvars`: A file where you will input your specific configuration values.

## File Details

### `main.tf`

- `google_bigquery_dataset`: Configures the BigQuery dataset.
- `google_bigquery_table`: Dynamically creates tables/views based on the `tables` variable.

### `variables.tf`

- `project_id`: Your GCP project ID.
- `view_dataset_id`: ID for the BigQuery dataset.
- `view_dataset_location`: Location for the dataset. Default is "US".
- `mc_tables`: A map containing the configuration for each BigQuery tables (assets, groups, preference_sets) associated with migration center (created by mc2bq tool).
- `cast_tables`: A map containing the configuration for the BigQuery table associated with cast file (created by [data-importer-function](https://github.com/bishtkomal/mc-cast-tf/blob/main/cloud-function-trigger/README.md#expected-results-from-terraform) )

### `terraform.tfvars`

- Provide your specific values for `project_id`, `view_dataset_id`, and the `view_dataset_location`.
- `mc_tables` has 3 dictionaries each associated with the migration center tables in bigquery viz. assets, groups, preference_sets. Provide all the information about each table in the associated dictionary having the same name. This means, all information for the table "assets" will be updated inside the dictionary named "assets".
- `cast_tables` has 1 dictionary named "AnalysisResults". Update this with information around the table containg cast data.

## Customization

To add more cast tables :
- Update the `cast_tables` map in `terraform.tfvars`.
- Each entry should include a dictionary with table details similar to the existing one.

## Usage Instructions

1. **Configure Terraform Variables**:
   - Open `terraform.tfvars`.
   - Update `project_id` with your GCP project ID.
   - Update `view_dataset_id` with your desired BigQuery dataset ID. This is the dataset you created in step 1.
   - Update `view_dataset_location` with location where your dataset containing the looker views will be created. Please note that this location should be the same as the location of the datasets created for migration center data and cast files.
   - Update `mc_tables` with information around all the migration center tables viz. assets, groups and preference sets. Update all values with correct project IDs, dataset IDs, table names.
   - Update `cast_tables` with information around the table which was created by the data-importer-function (created [here](https://github.com/bishtkomal/mc-cast-tf/blob/main/cloud-function-trigger/README.md#expected-results-from-terraform)

2. **Initialize Terraform**:
   - Open your command line interface.
   - Navigate to the directory containing your Terraform files.
   - Run `terraform init`. This command will download the necessary Terraform providers.

3. **Apply Terraform Configuration**:
   - Run `terraform apply`.
   - Review the plan and type `yes` to proceed with the creation of resources.

4. **Verify in GCP Console**:
   - Once Terraform successfully applies the configuration, verify the resources in your GCP console.

## Expected Results

### 1) Dataset
   - <dataset_id> : Will contain all the views mentioned below.
### 2) Views
   - <project_id>.<dataset_id>.CastReadiness_vw : Contains all data extracted from CAST Highlights tool after applying several transformation steps.
   - <project_id>.<dataset_id>.MigrationCenterInfra_vw : Contains all data extracted from Migration Center after applying several transformation steps.
   - <project_id>.<dataset_id>.McCastReadinessCombined_vw : Contains combined data from migration Center and CAST joined on the basis of groups. **This is the main view that will be exposed to Looker studio as a data source.**


