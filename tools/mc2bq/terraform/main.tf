// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.51.0"
    }
  }
}

locals {
  target_project = var.target_project != "" ? var.target_project : var.project
  mc2bq_cloud_run_image = var.mc2bq_cloud_run_image != "" ? var.mc2bq_cloud_run_image : "gcr.io/${var.project}/mc2bq:latest"
}

provider "google" {
  project = var.project
  region  = var.region
}

resource "google_service_account" "mc2bq_sync_trigger_sa" {
  account_id = "mc2bq-cloud-run-sync-trigger"
  display_name = "MC2BQ Cloud Run Sync Trigger"
  description  = "Service account created to trigger the Migration Center to BigQuery cloud run sync job"
  project      = var.project
}

resource "google_project_iam_member" "mc2bq_sync_trigger_sa_run_invoker" {
  project = var.project
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.mc2bq_sync_trigger_sa.email}"
}

resource "google_service_account" "mc2bq_sync_sa" {
  account_id = "mc2bq-cloud-run-sync"
  display_name = "MC2BQ Cloud Run Sync"
  description  = "Service account created for Migration Center to BigQuery cloud run sync job"
  project      = var.project
}

resource "google_project_iam_member" "mc2bq_cloud_run_bq_editor_binding" {
  project = var.project
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:${google_service_account.mc2bq_sync_sa.email}"
}

resource "google_project_iam_member" "mc2bq_cloud_run_bq_job_user_binding" {
  project = var.project
  role    = "roles/bigquery.jobUser"
  member  = "serviceAccount:${google_service_account.mc2bq_sync_sa.email}"
}

resource "google_project_iam_member" "mc2bq_cloud_run_mc_viewer_binding" {
  project = var.project
  role    = "roles/migrationcenter.viewer"
  member  = "serviceAccount:${google_service_account.mc2bq_sync_sa.email}"
}

resource "google_cloud_run_v2_job" "mc2bq_cloud_run_sync_job" {
  name = "mc2bq-sync"
  location = var.region

  template {
    parallelism = 1
    task_count = 1
    template {
      service_account = google_service_account.mc2bq_sync_sa.account_id
      timeout = "1800s" // 30m
      containers {
        image = local.mc2bq_cloud_run_image
        args = concat([
          "-force",
          "-target-project", local.target_project,
          "-region", var.region,
          var.project,
          var.dataset
        ], var.table_prefix == "" ? [] : [var.table_prefix])
      }
    }
  }

  lifecycle {
    ignore_changes = [
      launch_stage,
    ]
  }
}

resource "google_cloud_scheduler_job" "mc2bq_sync_scheduled_job" {
  name = "mc2bq-sync"
  description = "MC2BQ sync"
  schedule = var.mc2bq_sync_schedule
  time_zone = var.mc2bq_sync_schedule_timezone
  project = var.project
  region = var.region

  http_target {
    http_method = "POST"
    uri = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${google_cloud_run_v2_job.mc2bq_cloud_run_sync_job.project}/jobs/${google_cloud_run_v2_job.mc2bq_cloud_run_sync_job.name}:run"
    headers = {
      "User-Agent": "Google-Cloud-Scheduler"
    }

    oauth_token {
      service_account_email = google_service_account.mc2bq_sync_trigger_sa.email
      scope = "https://www.googleapis.com/auth/cloud-platform"
    }

  }
}
