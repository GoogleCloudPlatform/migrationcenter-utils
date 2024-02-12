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
