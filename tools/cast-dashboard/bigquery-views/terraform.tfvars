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
