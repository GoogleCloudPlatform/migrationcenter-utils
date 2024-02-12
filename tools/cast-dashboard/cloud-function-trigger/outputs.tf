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
