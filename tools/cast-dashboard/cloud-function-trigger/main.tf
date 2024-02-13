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

provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
}

resource "google_project_service" "pubsub_api" {
  service                    = "pubsub.googleapis.com"
  disable_dependent_services = true
  depends_on = [
    google_project_service.cloud_functions_api,
    google_project_service.cloud_build_api,
    google_project_service.artifact_registry_api,
  ]
}
resource "google_project_service" "cloud_functions_api" {
  service                    = "cloudfunctions.googleapis.com"
  disable_dependent_services = true
}

resource "google_project_service" "artifact_registry_api" {
  service                    = "artifactregistry.googleapis.com"
  disable_dependent_services = true
}

resource "google_project_service" "migration_center_api" {
  service                    = "rapidmigrationassessment.googleapis.com"
  disable_dependent_services = true
}

resource "google_project_service" "cloud_build_api" {
  service                    = "cloudbuild.googleapis.com"
  disable_dependent_services = true
}

resource "google_project_service" "cloud_logging_api" {
  service                    = "logging.googleapis.com"
  disable_dependent_services = true
}

resource "google_project_service" "cloud_run_admin_api" {
  service                    = "run.googleapis.com"
  disable_dependent_services = true
}
resource "google_pubsub_topic" "import_topic" {
  name = "import-topic"
}

resource "google_pubsub_topic" "assessment_topic" {
  name = "assessment-topic"
}

resource "google_storage_notification" "notification" {
  bucket         = var.bucket_name
  event_types    = ["OBJECT_FINALIZE"]
  payload_format = "JSON_API_V1"
  topic          = google_pubsub_topic.import_topic.id

  depends_on = [google_pubsub_topic_iam_binding.binding]
}

data "google_storage_project_service_account" "gcs_account" {
}

resource "google_pubsub_topic_iam_binding" "binding" {
  topic = google_pubsub_topic.import_topic.id
  role  = "roles/pubsub.publisher"
  members = [
    "serviceAccount:${data.google_storage_project_service_account.gcs_account.email_address}"
  ]
}

resource "google_cloudfunctions_function" "data_importer_function" {
  name                  = "data-to-bigquery-importer"
  runtime               = "python310" // Updated to 2nd gen
  available_memory_mb   = 128
  source_archive_bucket = var.bucket_name
  source_archive_object = "data_importer_source.zip"
  entry_point           = "import_csv_to_bigquery"

  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.import_topic.id
  }

  environment_variables = {
    BUCKET_NAME      = var.bucket_name
    BIGQUERY_DATASET = var.bigquery_dataset
    BIGQUERY_TABLE   = var.bigquery_table
  }

  depends_on = [
    google_project_service.artifact_registry_api,
    google_project_service.cloud_build_api,
    google_project_service.cloud_logging_api,
    google_project_service.cloud_run_admin_api
  ]
}

resource "google_storage_bucket_iam_member" "importer_bucket_access" {
  bucket = var.bucket_name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_cloudfunctions_function.data_importer_function.service_account_email}"
}

resource "google_storage_bucket_iam_member" "assessment_bucket_access" {
  bucket = var.bucket_name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_cloudfunctions_function.migration_assessment_function.service_account_email}"
}

resource "google_cloudfunctions_function" "migration_assessment_function" {
  name                  = "mc-create-grp-function"
  runtime               = "python310" // Updated to 2nd gen
  available_memory_mb   = 256
  source_archive_bucket = var.bucket_name
  source_archive_object = "migration_assessment_source.zip"
  entry_point           = "create_mc_group"

  environment_variables = {
    MIGRATIONCENTER_PATH = var.migrationcenter_path
    PROJECT_NAME         = var.gcp_project_id
  }

  event_trigger {
    event_type = "google.storage.object.finalize"
    resource   = "projects/${var.gcp_project_id}/buckets/${var.bucket_name}"
  }

  depends_on = [
    google_project_service.artifact_registry_api,
    google_project_service.cloud_build_api,
    google_project_service.cloud_logging_api,
    google_project_service.cloud_run_admin_api
  ]
}

resource "google_project_iam_member" "migration_assessment_sa" {
  project = var.gcp_project_id
  count   = length(var.sa_roles_ist)
  role    = var.sa_roles_ist[count.index]
  member  = "serviceAccount:${google_cloudfunctions_function.migration_assessment_function.service_account_email}"
  depends_on = [
    google_cloudfunctions_function.migration_assessment_function
  ]
}
resource "google_project_iam_member" "data_importer_sa" {
  project = var.gcp_project_id
  count   = length(var.sa_roles_ist)
  role    = var.sa_roles_ist[count.index]
  member  = "serviceAccount:${google_cloudfunctions_function.data_importer_function.service_account_email}"
  depends_on = [
    google_cloudfunctions_function.data_importer_function
  ]
}

resource "google_bigquery_dataset" "cast_dataset" {
  dataset_id = var.bigquery_dataset
  project    = var.gcp_project_id
  location   = var.bigquery_dataset_location

  access {
    role          = "OWNER"
    special_group = "projectOwners"
  }

  access {
    role          = "READER"
    special_group = "projectReaders"
  }

  access {
    role          = "WRITER"
    special_group = "projectWriters"
  }
}