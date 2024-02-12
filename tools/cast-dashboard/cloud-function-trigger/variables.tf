variable "gcp_project_id" {
  description = "The Google Cloud Platform project ID"
  type        = string
}

variable "gcp_region" {
  description = "The region where Google Cloud resources will be created"
  type        = string
}

variable "bucket_name" {
  description = "Name of the Google Cloud Storage bucket"
  type        = string
}

variable "bigquery_dataset" {
  description = "BigQuery dataset ID"
  type        = string
}

variable "bigquery_table" {
  description = "BigQuery table ID"
  type        = string
}

variable "migrationcenter_path" {
  description = "Migration Center path for the migration assessment function"
  type        = string
}

variable "sa_roles_ist" {
type =list(string)
default = ["roles/bigquery.jobUser","roles/bigquery.dataEditor", "roles/migrationcenter.admin"]
}

variable "bigquery_dataset_location" {
  description = "The region Google bigquery dataset will be created"
  type        = string
}