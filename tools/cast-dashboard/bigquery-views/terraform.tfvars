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

project_id = "<project-id>"
view_dataset_id = "mc_cast_looker_vws"
view_dataset_location = "US"
mc_tables = {
  "assets" = {
    project = "<project-id>"
    dataset = "migration_center"
    table   = "assets"
  },
  "groups" = {
    project = "<project-id>" 
    dataset = "migration_center"
    table   = "groups"
  },
  "preferences" = {
    project = "<project-id>" 
    dataset = "migration_center"
    table   = "preferences"
  }
}

cast_tables = {
  "AnalysisResults" = {
    project = "<project-id>"
    dataset = "cast_dataset"
    table   = "cast_analysis"
  },
  # "SurveyResults" = {
  #   project = "project-2" 
  #   dataset = "mcbq_tt"
  #   table   = "my_table_2"
  # }
}
