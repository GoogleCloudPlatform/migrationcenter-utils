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

resource "google_bigquery_dataset" "hsp_analysis_views" {
  dataset_id = var.view_dataset_id
  project    = var.project_id
  location   = var.view_dataset_location

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

# resource "google_bigquery_table" "bigquery_table" {
#   for_each   = var.tables
#   dataset_id = google_bigquery_dataset.hsp_analysis_views.dataset_id
#   project    = var.project_id
#   table_id   = each.key
#   schema     = each.value.schema

#   view {
#     query          = replace(each.value.query, "testproject-340909.Hsp_Analysis_Views", "${var.project_id}.${var.view_dataset_id}")
#     use_legacy_sql = false
#   }
# }

resource "google_bigquery_table" "migrationcenterinfra_vw" {
  dataset_id = google_bigquery_dataset.hsp_analysis_views.dataset_id
  project    = google_bigquery_dataset.hsp_analysis_views.project
  table_id   = "migrationcenterinfra_vw"
  schema     = "[{\"name\":\"Application\",\"type\":\"STRING\"},{\"name\":\"VMs\",\"type\":\"STRING\"},{\"name\":\"vCPUs\",\"type\":\"INTEGER\"},{\"name\":\"MemoryGBs\",\"type\":\"FLOAT\"},{\"name\":\"StorageGBs\",\"type\":\"FLOAT\"}]"
  view {
    query          = "SELECT grp.display_name as Application, machine_details.machine_name AS VMs, machine_details.core_count AS vCPUs, machine_details.memory_mb/1024 AS MemoryGBs, machine_details.disks.total_capacity_bytes/(1024*1024*1024) AS StorageGBs FROM `${var.mc_tables["assets"].project}.${var.mc_tables["assets"].dataset}.${var.mc_tables["assets"].table}` AS assets, UNNEST(assigned_groups) AS application INNER JOIN  `${var.mc_tables["groups"].project}.${var.mc_tables["groups"].dataset}.${var.mc_tables["groups"].table}` AS grp ON application = grp.name"
    use_legacy_sql = false
  }
}

resource "google_bigquery_table" "castreadiness_vw" {
  dataset_id = google_bigquery_dataset.hsp_analysis_views.dataset_id
  project    = google_bigquery_dataset.hsp_analysis_views.project
  table_id   = "castreadiness_vw"
  schema     = "[{\"mode\":\"NULLABLE\",\"name\":\"Application\",\"type\":\"STRING\"},{\"mode\":\"NULLABLE\",\"name\":\"BusinessUnits\",\"type\":\"STRING\"},{\"mode\":\"NULLABLE\",\"name\":\"BusinessValue\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"CloudReadyScore\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"Technologies\",\"type\":\"STRING\"},{\"mode\":\"NULLABLE\",\"name\":\"Software_Resiliency\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"Roadblocks\",\"type\":\"INTEGER\"},{\"mode\":\"NULLABLE\",\"name\":\"Lines_of_Code\",\"type\":\"INTEGER\"},{\"mode\":\"NULLABLE\",\"name\":\"DigitalReadiness\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"TechnicalDebtWeeks\",\"type\":\"FLOAT\"}]"
  view {
    query = "SELECT  Application,  Business_Units AS BusinessUnits,  BusinessValue,  CloudReady AS CloudReadyScore,  REPLACE(Technologies, \";\", \"\\n\") AS Technologies,  Software_Resiliency,  Roadblocks,  Lines_of_Code,  Digital_Readiness AS DigitalReadiness,  Technical_Debt__min__/10080 AS TechnicalDebtWeeks FROM   `${var.cast_tables["AnalysisResults"].project}.${var.cast_tables["AnalysisResults"].dataset}.${var.cast_tables["AnalysisResults"].table}` WHERE Business_Units IS NOT NULL"

    use_legacy_sql = false
  }
}

resource "google_bigquery_table" "mccastreadinesscombined_vw" {
  dataset_id = google_bigquery_dataset.hsp_analysis_views.dataset_id
  project    = google_bigquery_dataset.hsp_analysis_views.project
  table_id   = "mccastreadinesscombined_vw"
  schema     = "[{\"mode\":\"NULLABLE\",\"name\":\"Application\",\"type\":\"STRING\"},{\"mode\":\"NULLABLE\",\"name\":\"BusinessUnits\",\"type\":\"STRING\"},{\"mode\":\"NULLABLE\",\"name\":\"BusinessValue\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"VMs\",\"type\":\"INTEGER\"},{\"mode\":\"NULLABLE\",\"name\":\"vCPUs\",\"type\":\"INTEGER\"},{\"mode\":\"NULLABLE\",\"name\":\"MemoryGBs\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"StorageGBs\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"CloudReadyScore\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"Technologies\",\"type\":\"STRING\"},{\"mode\":\"NULLABLE\",\"name\":\"SoftwareResiliency\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"Roadblocks\",\"type\":\"INTEGER\"},{\"mode\":\"NULLABLE\",\"name\":\"LinesOfCode\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"DigitalReadiness\",\"type\":\"FLOAT\"},{\"mode\":\"NULLABLE\",\"name\":\"TechnicalDebtWeeks\",\"type\":\"FLOAT\"}]"
  view {
    query = "SELECT\n  CastR.Application,\n  REPLACE(CastR.BusinessUnits, \";\", \"\\n\") as BusinessUnits,\n  AVG(CastR.BusinessValue) as BusinessValue,\n  COUNT(McInfra.VMs) AS VMs,\n  SUM(McInfra.vCPUs) AS vCPUs,\n  SUM(McInfra.MemoryGBs) AS MemoryGBs,\n  SUM(McInfra.StorageGBs) AS StorageGBs,\n  AVG(CastR.CloudReadyScore) AS CloudReadyScore,\n  CastR.Technologies,\n  AVG(CastR.Software_Resiliency) as SoftwareResiliency,\n  CastR.Roadblocks,\n  AVG(CastR.Lines_of_Code) AS LinesOfCode,\n  AVG(CastR.DigitalReadiness) AS DigitalReadiness,\n  AVG(CastR.TechnicalDebtWeeks) AS TechnicalDebtWeeks\nFROM\n  `${var.project_id}.${var.view_dataset_id}.migrationcenterinfra_vw` AS McInfra\nINNER JOIN\n  `${var.project_id}.${var.view_dataset_id}.castreadiness_vw` AS CastR\nON\n  McInfra.Application = CastR.Application\nGROUP BY\n  CastR.Application,\n  CastR.BusinessUnits,\n  CastR.Roadblocks,\n  CastR.Technologies"

    use_legacy_sql = false
  }
  depends_on = [
    google_bigquery_table.castreadiness_vw,
    google_bigquery_table.migrationcenterinfra_vw
    ]
}
