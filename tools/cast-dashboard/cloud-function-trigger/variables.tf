# Copyright 2023 Google LLC All Rights Reserved.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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