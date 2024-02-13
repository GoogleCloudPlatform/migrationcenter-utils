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

output "bucket_name" {
  value       = var.bucket_name
  description = "Name of the Google Cloud Storage bucket where CSV files are uploaded."
}

output "import_function_name" {
  value       = google_cloudfunctions_function.data_importer_function.name
  description = "Name of the data importer Cloud Function."
}

output "bigquery_dataset" {
  value       = var.bigquery_dataset
  description = "BigQuery dataset ID where the CSV files will be loaded."
}

output "bigquery_table" {
  value       = var.bigquery_table
  description = "BigQuery table ID where the CSV files will be loaded."
}

output "pubsub_import_topic" {
  value       = google_pubsub_topic.import_topic.name
  description = "Pub/Sub topic for the data importer function."
}

output "pubsub_assessment_topic" {
  value       = google_pubsub_topic.assessment_topic.name
  description = "Pub/Sub topic for the migration assessment function."
}

output "assessment_function_name" {
  value       = google_cloudfunctions_function.migration_assessment_function.name
  description = "Name of the migration assessment Cloud Function."
}
