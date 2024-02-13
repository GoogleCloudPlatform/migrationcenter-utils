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

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "view_dataset_id" {
  description = "BigQuery Dataset ID where all views will be created"
  type        = string
}

variable "view_dataset_location" {
  description = "The location for the BigQuery dataset"
  default     = "US"
}

variable "mc_tables" {
  description = "A map of tables with data from migration center "
  type = map(object({
    project = string
    dataset = string
    table = string
  }))
}

variable "cast_tables" {
  description = "A map of tables with data from migration center "
  type = map(object({
    project = string
    dataset = string
    table = string
  }))
}
