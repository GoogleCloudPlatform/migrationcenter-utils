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

variable "project" {}
variable "region" {
  default = "us-central1"
}

variable "target_project" {
  default = "" // defaults to project if undefined
}

variable "mc2bq_cloud_run_image" {
  default = "" // defaults to gcr.io/<project>/mc2bq:latests if undefined
}

variable "dataset" {
  default = "migration_center"
}

variable "table_prefix" {
  default = ""
}

variable "mc2bq_sync_schedule" {
  default = "0 0 * * *" // Daily at midnight
}

variable "mc2bq_sync_schedule_timezone" {
  default = "Etc/UTC"
}
