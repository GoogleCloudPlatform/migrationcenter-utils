#################################################################
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Migration Pricing Reports C2C Data Import

# v0.2
# Google
# amarcum@google.com
#################################################################

import pandas as pd
import urllib
import gspread
import csv
import datetime
from oauth2client.service_account import ServiceAccountCredentials
from google.cloud import bigquery
import google.auth
from gspread_formatting import *
import argparse
import time
import os
import json

version = "v0.2"
datetime = str(datetime.datetime.now().strftime("%Y-%m-%d %H:%M"))
username = os.environ['USER']
if username == 'root':
    print("User root not allowed to run this application! Exiting...")
    exit()

# Order to sort worksheets and new names to use
f = open('settings.json', )
settings_file = json.load(f)
mc_names = settings_file["mc_names"]
mc_column_names = settings_file["mc_column_names"]
refresh_data_sources_body = settings_file["refresh_data_sources"]
f.close()

default_mc_looker_template_id = "421c8150-e7ad-4190-b044-6a18ecdbd391"
default_cur_looker_template_id = "c4e0ccbc-907a-4bc4-85f1-1711ee47c345"


# Check number of rows & columns in CSV file
def check_csv_size(mc_reports_directory):
    print("Checking CSV sizes...")
    mc_file_list = os.listdir(mc_reports_directory)
    if any('.csv' in file for file in mc_file_list):
        for file in mc_file_list:
            if file.endswith(".csv"):
                file_fullpath = (mc_reports_directory + "/" + file)
                csv_data = pd.read_csv(file_fullpath, nrows=1)
                try:
                    number_of_columns = len(csv_data.values[0].tolist())
                except:
                    number_of_columns = 0

                with open(file_fullpath, "rb") as f:
                    number_of_rows = sum(1 for _ in f)

                total_cells = number_of_rows * number_of_columns
                if total_cells > 5000000:
                    print(file + " exceeds the 5 million cell Google Sheets limit (" + str(
                        total_cells) + ") and therefor cannot be imported through the Google Sheets API. Consider using the -b & -n argument to import into Big Query & Sheets instead.")
                    exit()
    else:
        print("No CSV files found in " + mc_reports_directory + "! Exiting!")
        exit()


# Create Initial Google Sheets
def create_google_sheets(customer_name, sheets_email_addresses, service_account_key, sheets_id):
    if sheets_id == "":
        # print("\nCreating new Google Sheets...")
        test = 0
    else:
        print("\nUpdating Google Sheets: " + sheets_id)

    scope = ['https://www.googleapis.com/auth/drive', 'https://www.googleapis.com/auth/spreadsheets']
    sheets_title = ("Migration Center Pricing Report: " + customer_name + ' - ' + datetime)

    # Use provided Google Service Account Key, otherwise try to use gcloud auth key to authenticate

    credentials = google_auth(service_account_key, scope)

    client = gspread.authorize(credentials)

    # Depending on CLI Args - create new sheet or update existing
    if sheets_id == '':
        spreadsheet = client.create(sheets_title)
        spreadsheet = client.open(sheets_title)

    else:
        spreadsheet = client.open_by_key(sheets_id)

    # If any emails are provided, share sheets with them
    for shared_user in sheets_email_addresses:
        spreadsheet.share(shared_user, perm_type='user', role='writer', notify=False)

    return spreadsheet, credentials


def generate_pie_table_request(spreadsheet, chart_title, ref_column, value_column, position_data):
    # Google Sheets Charts API: https://developers.google.com/sheets/api/samples/charts

    f = open('settings.json', )
    template_file = json.load(f)
    new_pie_chart_request = template_file["pie_chart_request"]
    f.close()

    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["title"] = chart_title
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["pieChart"]["domain"]["sourceRange"]["sources"][
        0]["sheetId"] = spreadsheet
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["pieChart"]["domain"]["sourceRange"]["sources"][
        0]["startColumnIndex"] = ref_column
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["pieChart"]["domain"]["sourceRange"]["sources"][
        0]["endColumnIndex"] = ref_column + 1

    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["pieChart"]["series"]["sourceRange"]["sources"][
        0]["sheetId"] = spreadsheet
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["pieChart"]["series"]["sourceRange"]["sources"][
        0]["startColumnIndex"] = value_column
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["spec"]["pieChart"]["series"]["sourceRange"]["sources"][
        0]["endColumnIndex"] = value_column + 1

    new_pie_chart_request["requests"][0]["addChart"]["chart"]["position"]["overlayPosition"]["anchorCell"][
        "sheetId"] = spreadsheet
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["position"]["overlayPosition"]["anchorCell"][
        "columnIndex"] = position_data[0]
    new_pie_chart_request["requests"][0]["addChart"]["chart"]["position"]["overlayPosition"]["anchorCell"]["rowIndex"] = \
        position_data[1]

    return new_pie_chart_request


def generate_repeat_cell_formula_request(sheet_id, formula, start_column, start_row):
    # "startColumnIndex": 2 and "endColumnIndex": 3 of range means the column "C".
    # "startRowIndex": 1 of range and no endRowIndex means that the formula is put from the row 2 to end of row.

    # Currently only support 1 column
    end_column = start_column + 1

    body = {
        # "requests": [
        #     {
        "repeatCell": {
            "cell": {
                "userEnteredValue": {
                    "formulaValue": formula
                }
            },
            "range": {
                "sheetId": sheet_id,
                "startColumnIndex": start_column,
                "endColumnIndex": end_column,
                "startRowIndex": start_row
            },
            "fields": "userEnteredValue.formulaValue"
        }
    }
    #     ]
    # }
    return body


def autosize_worksheet(sheet_id, first_col, last_col):
    # Autoresize Worksheet - Body values
    body = {
        "requests": [
            {
                "autoResizeDimensions": {
                    "dimensions": {
                        "sheetId": sheet_id,
                        "dimension": "COLUMNS",
                        "startIndex": first_col,  # Please set the column index.
                        "endIndex": last_col  # Please set the column index.
                    }
                }
            }
        ]
    }

    return body


def apply_conditional_color_rule(sheet_id, grid_range, boolean_condition, boolean_value, colors):
    # Set up conditional rules (Red/Green) to GCP Overview Differences
    rule = ConditionalFormatRule(
        ranges=[GridRange.from_a1_range(grid_range, sheet_id)],
        booleanRule=BooleanRule(
            condition=BooleanCondition(boolean_condition, [boolean_value]),
            format=CellFormat(textFormat=textFormat(foregroundColor=Color(colors[0], colors[1], colors[2])))

        )
    )

    sheet_rules = get_conditional_format_rules(sheet_id)
    sheet_rules.append(rule)
    sheet_rules.save()


# Create API Request to Connect BQ Table to Google Sheets
def connect_bq_to_sheets(gcp_project_id, bq_dataset_name, bq_table):
    body = {
        "requests": [
            {
                "addDataSource": {
                    "dataSource": {
                        "spec": {
                            "bigQuery": {
                                "projectId": gcp_project_id,
                                "tableSpec": {
                                    "tableProjectId": gcp_project_id,
                                    "datasetId": bq_dataset_name,
                                    "tableId": bq_table
                                }
                            }
                        }
                    }
                }
            }
        ]
    }

    return body


def generate_protect_sheet_request(sheet_id):
    body = {
        "requests": [
            {
                "addProtectedRange": {
                    "protectedRange": {
                        "range": {
                            "sheetId": sheet_id,
                        },
                        "warningOnly": False
                    }
                }
            }
        ]
    }

    return body


# Create Pivot table with sums for Google Sheets
def generate_pivot_table_request(source, data_source, row_col, value_col, location_spreadsheet,
                                 pivot_table_location, summarize_function, row_name, row_col_2nd, row_name_2nd,
                                 value_name, value_col_2nd, value_name_2nd, filter_col, show_diff, row_col_3rd,
                                 row_name_3rd, row_col_4th, row_name_4th, row_col_5th, row_name_5th, value_col_3rd,
                                 value_name_3rd, value_col_4th, value_name_4th, value_col_5th, value_name_5th,
                                 row_col_6th, row_name_6th):
    # Google Sheets Pivot Table API: https://developers.google.com/sheets/api/samples/pivot-tables
    f = open('settings.json', )
    template_file = json.load(f)
    new_pivot_table_request = template_file["pivot_table_request"]
    f.close()

    if source == "BQ":
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
            "dataSourceId"] = data_source[0]
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
            "dataSourceId"] = data_source[0]

        # If defined, add a 2nd column source for the Pivot Table
        if row_col_2nd is not None:

            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
                "dataSourceColumnReference"] = {}
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
                "dataSourceColumnReference"]["name"] = row_col

            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"].append(
                {})
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "showTotals"] = False
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "sortOrder"] = "DESCENDING"
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "valueBucket"] = {}
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "dataSourceColumnReference"] = {}
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "dataSourceColumnReference"]["name"] = row_col_2nd

            if row_col_3rd is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append({})
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "valueBucket"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "dataSourceColumnReference"]["name"] = row_col_3rd

            if row_col_4th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append({})
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "valueBucket"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "dataSourceColumnReference"]["name"] = row_col_4th

            if row_col_5th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append({})
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "valueBucket"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "dataSourceColumnReference"]["name"] = row_col_5th

            if row_col_6th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append({})
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "valueBucket"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "dataSourceColumnReference"]["name"] = row_col_6th

        else:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
                "dataSourceColumnReference"]["name"] = row_col

        # If defined, add a 2nd column values for the Pivot Table
        if value_col_2nd is not None:

            if value_name is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    0]["name"] = value_name
            else:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    0]["name"] = "Total"

            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][0][
                "dataSourceColumnReference"] = {}
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][0][
                "dataSourceColumnReference"]["name"] = value_col

            # 2nd Values Column
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                "values"].append({})
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][1][
                "summarizeFunction"] = "SUM"

            if value_name_2nd is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    1]["name"] = value_name_2nd
            else:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    1]["name"] = "2nd Total"

            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][1][
                "dataSourceColumnReference"] = {}
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][1][
                "dataSourceColumnReference"]["name"] = value_col_2nd

            if value_col_3rd is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})
                if value_name_3rd is not None:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][2]["name"] = value_name_3rd
                else:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][2]["name"] = "3rd Total"

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["dataSourceColumnReference"]["name"] = value_col_3rd
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["summarizeFunction"] = "SUM"

            if value_col_4th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})
                if value_name_4th is not None:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][3]["name"] = value_name_4th
                else:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][3]["name"] = "4th Total"

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["dataSourceColumnReference"]["name"] = value_col_4th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["summarizeFunction"] = "SUM"

            if value_col_5th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})
                if value_name_5th is not None:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][4]["name"] = value_name_5th
                else:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][4]["name"] = "5th Total"

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    4]["dataSourceColumnReference"] = {}
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    4]["dataSourceColumnReference"]["name"] = value_col_5th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    4]["summarizeFunction"] = "SUM"

            if show_diff is True:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["summarizeFunction"] = "SUM"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["name"] = "Cost Difference"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["formula"] = "=(GCP_Cost - Source_Cost)"

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["summarizeFunction"] = "SUM"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["name"] = "% Difference"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["formula"] = "=((GCP_Cost - Source_Cost) / Source_Cost)"

        else:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][0][
                "dataSourceColumnReference"]["name"] = value_col

            if show_diff is True:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    1]["summarizeFunction"] = "SUM"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    1]["name"] = "Cost Difference"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    1]["formula"] = "=(GCP_Cost - Source_Cost)"

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["summarizeFunction"] = "SUM"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["name"] = "% Difference"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["formula"] = "=((GCP_Cost - Source_Cost) / Source_Cost)"

        if filter_col is None:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][
                0][
                "dataSourceColumnReference"]["name"] = value_col
        else:
            if filter_col != "CONTAINS":
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0][
                    "dataSourceColumnReference"]["name"] = filter_col
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0][
                    "filterCriteria"]["condition"]["type"] = "NOT_BLANK"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0][
                    "filterCriteria"]["condition"]["values"] = []
            else:

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0][
                    "dataSourceColumnReference"]["name"] = "Description"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0][
                    "filterCriteria"]["condition"]["type"] = "TEXT_CONTAINS"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0][
                    "filterCriteria"]["condition"]["values"] = {"userEnteredValue": "Compute Engine"}

    if source == "SHEETS":
        # Clean up template table and remove BQ references
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"].pop("dataSourceId",
                                                                                                        None)
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"].pop("rows", None)

        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"].pop("values", None)

        # Create Filter for Pivot Table - default is to not show anything less than zero
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"].pop('filterSpecs',
                                                                                                        None)
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"] = []
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
            "filterSpecs"].append({})

        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][0][
            "filterCriteria"] = {}
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][0][
            "filterCriteria"]["visibleByDefault"] = True
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][0][
            "filterCriteria"]["condition"] = {}

        if filter_col is None:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][
                0]["columnOffsetIndex"] = value_col
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][
                0]["filterCriteria"]["condition"]["type"] = "NUMBER_GREATER"
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][
                0]["filterCriteria"]["condition"]["values"] = []
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][
                0]["filterCriteria"]["condition"]["values"].append({})
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["filterSpecs"][
                0]["filterCriteria"]["condition"]["values"][0]["userEnteredValue"] = "0"
        else:
            if filter_col != "CONTAINS":
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0]["columnOffsetIndex"] = filter_col
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0]["filterCriteria"]["condition"]["type"] = "NOT_BLANK"
            else:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0]["columnOffsetIndex"] = 6
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0]["filterCriteria"]["condition"]["type"] = "TEXT_CONTAINS"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "filterSpecs"][
                    0]["filterCriteria"]["condition"]["values"] = {"userEnteredValue": "Compute Engine"}

        # Change templated Pivot Table source to use cells from worksheet
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["source"] = {}
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["source"][
            "sheetId"] = data_source[0]
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["source"][
            "startRowIndex"] = 0
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["source"][
            "startColumnIndex"] = 0
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["source"][
            "endColumnIndex"] = data_source[1]
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["source"][
            "endRowIndex"] = data_source[2]

        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"] = []
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"].append({})

        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
            "sourceColumnOffset"] = row_col
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
            "showTotals"] = False
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
            "sortOrder"] = "DESCENDING"
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][0][
            "valueBucket"] = {}

        # If defined, add a 2nd column source for the Pivot Table
        if row_col_2nd is not None:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"].append(
                {})

            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "sourceColumnOffset"] = row_col_2nd
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "showTotals"] = False
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "sortOrder"] = "DESCENDING"
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][1][
                "valueBucket"] = {}

            if row_col_3rd is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append(
                    {})

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "sourceColumnOffset"] = row_col_3rd
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][2][
                    "valueBucket"] = {}

            if row_col_4th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append(
                    {})

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "sourceColumnOffset"] = row_col_4th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][3][
                    "valueBucket"] = {}

            if row_col_5th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append(
                    {})

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "sourceColumnOffset"] = row_col_5th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][4][
                    "valueBucket"] = {}

            if row_col_6th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "rows"].append(
                    {})

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "sourceColumnOffset"] = row_col_6th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "showTotals"] = False
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "sortOrder"] = "DESCENDING"
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["rows"][5][
                    "valueBucket"] = {}

        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"] = []
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"].append({})

        if value_name is not None:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][0][
                "name"] = value_name

        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][0][
            "sourceColumnOffset"] = value_col
        new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][0][
            "summarizeFunction"] = summarize_function

        if value_col_2nd is not None:
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                "values"].append({})

            if value_name_2nd is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    1]["name"] = value_name_2nd

            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][1][
                "sourceColumnOffset"] = value_col_2nd
            new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][1][
                "summarizeFunction"] = summarize_function

            if value_col_3rd is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})

                if value_name_3rd is not None:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][2]["name"] = value_name_3rd

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["sourceColumnOffset"] = value_col_3rd
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    2]["summarizeFunction"] = summarize_function

            if value_col_4th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})

                if value_name_4th is not None:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][3]["name"] = value_name_4th

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["sourceColumnOffset"] = value_col_4th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    3]["summarizeFunction"] = summarize_function

            if value_col_5th is not None:
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                    "values"].append({})

                if value_name_5th is not None:
                    new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"][
                        "values"][4]["name"] = value_name_5th

                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    4]["sourceColumnOffset"] = value_col_5th
                new_pivot_table_request["requests"][0]["updateCells"]["rows"][0]["values"][0]["pivotTable"]["values"][
                    4]["summarizeFunction"] = summarize_function

    new_pivot_table_request["requests"][0]["updateCells"]["start"]["sheetId"] = location_spreadsheet
    new_pivot_table_request["requests"][0]["updateCells"]["start"]["rowIndex"] = pivot_table_location[1]
    new_pivot_table_request["requests"][0]["updateCells"]["start"]["columnIndex"] = pivot_table_location[0]

    return new_pivot_table_request


def generate_mc_sheets(spreadsheet, worksheet_names, data_source_type, data_source, unmapped_data_worksheet):
    exec_overview_worksheets_name = "Executive Overview"
    gcp_overview_worksheets_name = "GCP Detailed Overview"
    unmapped_worksheets_name = "AWS Unmapped Overview"
    gcp_discounts_worksheets_name = "GCP Discounts"

    # Create Executive Overview Worksheet in Sheets
    exec_overview_worksheet = spreadsheet.add_worksheet(exec_overview_worksheets_name, 60, 25)
    exec_overview_worksheet_id = exec_overview_worksheet._properties['sheetId']

    # Create GCP Overview Worksheet in Sheets
    gcp_overview_worksheet = spreadsheet.add_worksheet(gcp_overview_worksheets_name, 125, 25)
    gcp_overview_worksheet_id = gcp_overview_worksheet._properties['sheetId']

    # Create AWS Unmapped Worksheet in Sheets
    unmapped_worksheet = spreadsheet.add_worksheet(unmapped_worksheets_name, 60, 25)
    unmapped_worksheet_id = unmapped_worksheet._properties['sheetId']

    # Create GCP Discounts Worksheet in Sheets
    gcp_discounts_worksheet = spreadsheet.add_worksheet(gcp_discounts_worksheets_name, 300, 25)
    gcp_discounts_worksheet_id = gcp_discounts_worksheet._properties['sheetId']

    # Create Machine Type Overview Worksheet in Sheets
    mt_overview_worksheet = spreadsheet.add_worksheet("Machine Type Overview", 300, 25)
    mt_overview_worksheet_id = mt_overview_worksheet._properties['sheetId']

    # Create Storage Overview Worksheet in Sheets - CURRENTLY DISABLED
    # storage_overview_worksheet = spreadsheet.add_worksheet("Storage Overview", 60, 25)
    # storage_overview_worksheet_id = storage_overview_worksheet._properties['sheetId']

    # Create DB Overview Worksheet in Sheets - CURRENTLY DISABLED
    # db_overview_worksheet = spreadsheet.add_worksheet("Database Overview", 60, 25)
    # db_overview_worksheet_id = db_overview_worksheet._properties['sheetId']

    if data_source_type == "BQ":
        unmapped_data_column_formula = f"=SUM(\'{unmapped_data_worksheet}\'!lineItem_UnblendedCost)"
    elif data_source_type == "SHEETS":
        unmapped_data_column_formula = f"=SUM(\'{unmapped_data_worksheet}\'!L2:L)"

    exec_overview_worksheet.batch_update([
        {
            'range': "A1:A3",
            'values': [["AWS Spend (GCP Matched)"], ["AWS Spend (Unmatched)"], ["AWS Total Spend"]],
        }, {
            'range': "B1:B3",
            'values': [["=SUM(E2:E)"], [unmapped_data_column_formula], ["=B1+B2"]],
        },
        {
            'range': "A5:A7",
            'values': [["GCP Spend (AWS Matched)"], ["GCP Cost Difference"], ["GCP Percent Difference"]],
        }, {
            'range': "B5:B7",
            'values': [["=SUM(F2:F)"], ["=if($B$3=0, \"N/A\", B5-B1)"], ["=if($B$3=0, \"N/A\", B6/B1)"]],
        },
        {
            'range': "G1:I1",
            'values': [
                ["% of AWS Total Spend", "GCP Cost Difference", "GCP Percent Difference"]],
        }
    ]
        , value_input_option="USER_ENTERED"
    )

    # Insert Formulas off of pivot tables for above columns
    exec_overview_formula_json_request = {"requests":
        [
            generate_repeat_cell_formula_request(exec_overview_worksheet_id,
                                                 "=IF(ISBLANK($E2), \"\", IF($E2=0, \"N/A\", $E2/$B$3))", 6,
                                                 1),
            generate_repeat_cell_formula_request(exec_overview_worksheet_id, "=IF(ISBLANK($E2), \"\", IF($E2=0, \"N/A\", $F2-$E2))", 7,
                                                 1),
            generate_repeat_cell_formula_request(exec_overview_worksheet_id,
                                                 "=IF(ISBLANK($E2), \"\", if($F2<=0,\"No Google Cost\",IF($E2=0, \"N/A\",($F2-$E2)/$E2)))",
                                                 8,
                                                 1)
        ]
    }

    response = spreadsheet.batch_update(exec_overview_formula_json_request)

    gcp_overview_worksheet.batch_update([{
        'range': "D1:E1",
        'values': [
            ["GCP Cost Difference", "GCP Percent Difference"]],
    }]
    )

    # GCP Details - GCP Cost Difference
    gcp_overview_formula_json_request = {"requests":
        [
            generate_repeat_cell_formula_request(gcp_overview_worksheet_id,
                                                 "=IF(ISBLANK($B2), \"\", IF($B2=0, \"N/A\", $C2-$B2))", 3, 1),
            generate_repeat_cell_formula_request(gcp_overview_worksheet_id,
                                                 "=IF(ISBLANK($B2), \"\", IF($B2=0, \"N/A\", if($C2<=0,\"No Google Cost\",($C2-$B2)/$B2)))",
                                                 4,
                                                 1),
        ]
    }

    response = spreadsheet.batch_update(gcp_overview_formula_json_request)

    # GCP % Discounts & GCP Discounted Price
    gcp_discounts_formula_json_request = {"requests":
        [
            # generate_repeat_cell_formula_request(gcp_discounts_worksheet_id, "=IF(ISBLANK($E3), \"\", $D3+$E3)", 5, 2),
            generate_repeat_cell_formula_request(gcp_discounts_worksheet_id, "=IF(ISBLANK($F3), \"\", 0)", 6, 2),
            generate_repeat_cell_formula_request(gcp_discounts_worksheet_id,
                                                 "=IF(ISBLANK($F3), \"\", if($F3<=0,0,if($E3=$F3,((1-$G3)*$F3),((1-$G3)*($F3-$D3)))))",
                                                 7,
                                                 2),
            generate_repeat_cell_formula_request(gcp_discounts_worksheet_id, "=IF(ISBLANK($F3), \"\", $D3 + $H3)", 8,
                                                 2),

        ]
    }
    response = spreadsheet.batch_update(gcp_discounts_formula_json_request)

    gcp_discounts_worksheet.batch_update([
        {
            'range': "A1",
            'values': [
                ["GCP Services Discounts"]],
        },
        {
            'range': "G2:I2",
            'values': [
                # ["GCP Total", "GCP Discount %", "GCP Discounted Price", "GCP Discounted Total"]],
                ["GCP Discount %", "GCP Discounted Price", "GCP Discounted Total"]],

        },
        {
            'range': "N2",
            'values': [
                ["* - Discount is only applied to Infra Cost"]],
        },
        {
            'range': "K2:K4",
            'values': [
                ["GCP Services Total"], ["GCP Services Discounted Total"], ["GCP Services Total w/ Discounts"]],
        },
        {
            'range': "L2:L4",
            'values': [
                ["=SUM(F3:F)"], ["=(SUM(F3:F) - SUM(H3:H))"], ["=SUM(H3:H) + SUM(D3:D)"]],
        }

    ], value_input_option="USER_ENTERED"
    )

    # Machine Type Overview Page
    mt_overview_formula_json_request = {"requests":
        [
            generate_repeat_cell_formula_request(mt_overview_worksheet_id,
                                                 "=IF(ISBLANK($H2), \"\", if($H2=0, \"N/A\", $K2-$H2))", 11, 1),
            generate_repeat_cell_formula_request(mt_overview_worksheet_id,
                                                 "=IF(ISBLANK($H2), \"\", if($H2=0, \"N/A\", ($K2-$H2)/$H2))", 12, 1),

        ]
    }
    response = spreadsheet.batch_update(mt_overview_formula_json_request)

    mt_overview_worksheet.batch_update([
        {
            'range': "L1:M1",
            'values': [
                ["GCP Cost Difference", "GCP Percent Difference"]],
        }

    ], value_input_option="USER_ENTERED"
    )

    # Add Cost sums to Overview Worksheet. Filter on GCP Cost column being greater than 0.
    pivot_table_location = [
        0,  # Column A
        0  # Row 1
    ]

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
        data_row_col = "GCP_Service"
        data_value_col = "Source_Cost"
        data_value_2nd_col = "GCP_Cost"
        filter_column = "Source_Cost"
    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]
        data_row_col = 5  # Data, Column F, GCP_Service
        data_value_col = 19  # Data, Column X, Source_Cost
        data_value_2nd_col = 23  # Data, Column T, GCP_Cost
        filter_column = 19  # Data, Column T, Source_Cost

    value_name = "AWS Cost"
    value_name_2nd = "GCP Cost"

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     gcp_overview_worksheet_id,
                                     pivot_table_location, "SUM", None, None, None, value_name,
                                     data_value_2nd_col,
                                     value_name_2nd, filter_column, False, None, None, None, None, None, None, None,
                                     None, None, None, None, None, None, None
                                     ),

    )

    # Add Region Breakdown to Overview Worksheet. Filter on GCP Cost column being greater than 0.
    pivot_table_location = [
        6,  # Column G
        0  # Row 1
    ]

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
        data_row_col = "Region"
        data_row_col_2nd = "GCP_Service"
        data_value_col = "GCP_Cost"
        filter_column = "Region"

    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]
        data_row_col = 7  # Data, Column H, Region
        data_row_col_2nd = 5  # Data, Column F, GCP_Service
        data_value_col = 23  # Data, Column X, GCP Cost
        filter_column = 7  # Data, Column H, Region

    # Add Instance Region Cost
    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     gcp_overview_worksheet_id,
                                     pivot_table_location, "SUM", None, data_row_col_2nd, None, "GCP Cost", None, None,
                                     filter_column,
                                     False, None, None, None, None, None, None, None, None, None, None, None, None,
                                     None, None
                                     ))

    # Add Instance Cost to Overview Worksheet. Filter on Destination_Shape column being not None.
    pivot_table_location = [
        10,  # Column K
        0  # Row 1
    ]

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
        data_row_col = "Region"
        data_value_col = "GCP_Cost"
        data_row_col_2nd = "Destination_Shape"
        filter_col = "Destination_Shape"

    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]
        data_row_col = 7  # Data, Column H, Region
        data_value_col = 23  # Data, Column X, GCP Cost
        data_row_col_2nd = 10  # Data, Column K, Destination Shape
        filter_col = 10  # Data, Column K, Destination Shape

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     gcp_overview_worksheet_id,
                                     pivot_table_location, "SUM", None, data_row_col_2nd, None, "GCP Cost", None, None,
                                     filter_col, False, None, None, None, None, None, None, None, None, None, None,
                                     None, None, None, None
                                     ))

    # Exec Overview Pivot Table
    pivot_table_location = [
        3,  # Column D
        0  # Row 1
    ]

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]

    # data_row_col_name = "Source_Product"
    # data_value_col_name = "Source_Cost"
    # data_value_2nd_col_name = "GCP_Cost"
    value_name = "AWS Cost"
    value_name_2nd = "GCP Cost"

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
        data_row_col = "Source_Product"
        data_value_col = "Source_Cost"
        data_row_col_2nd = "GCP_Cost"
        filter_column = "Source_Cost"
    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]
        data_row_col = 3  # Data, Column D, Source_Product
        data_value_col = 19  # Data, Column T, Source_Cost
        data_row_col_2nd = 23  # Data, Column K, GCP_Cost
        filter_column = 19  # Data, Column T, Source_Cost

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     exec_overview_worksheet_id,
                                     pivot_table_location, "SUM", None, None, None, value_name,
                                     data_row_col_2nd,
                                     value_name_2nd, filter_column, False, None, None, None, None, None, None, None,
                                     None, None, None, None, None, None, None
                                     ),

    )

    # Add GCP Discounts to Discounts worksheet.
    pivot_table_location = [
        0,  # Column A
        1  # Row 2
    ]

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
        data_row_col = "GCP_Service"
        data_row_col_2nd = "Destination_Series"
        data_row_col_3rd = "Description"
        data_value_col = "OS_Licenses_Cost"
        data_value_col_2nd = "Infra_Cost"
        data_value_col_3rd = "GCP_Cost"

        filter_column = "GCP_Service"
    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]
        data_row_col = 5  # Data, Column F, GCP_Service
        data_row_col_2nd = 8  # Data, Column I, Destination_Series
        data_row_col_3rd = 6  # Data, Column G, Description
        data_value_col = 22  # Data, Column W, OS_Licenses_Cost
        data_value_col_2nd = 21  # Data, Column V, Infra_Cost
        data_value_col_3rd = 23  # Data, Column X, GCP_Cost

        filter_column = 5  # Data, Column F, GCP_Service

    value_name = "License Cost"
    value_name_2nd = "Infra Cost"
    value_name_3rd = "GCP Total Cost"

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     gcp_discounts_worksheet_id,
                                     pivot_table_location, "SUM", None, data_row_col_2nd, None, value_name,
                                     data_value_col_2nd, value_name_2nd, filter_column, False, data_row_col_3rd, None,
                                     None, None, None, None, data_value_col_3rd, value_name_3rd, None, None, None, None,
                                     None, None
                                     ),

    )

    # Add Cost sums to Overview Worksheet. Filter on GCP Cost column being greater than 0.
    pivot_table_location = [
        0,  # Column A
        0  # Row 1
    ]

    if data_source_type == "BQ":
        data_source_id = [data_source[0]]
        data_row_col = "Region"
        data_row_col_2nd = "Source_Shape"
        data_row_col_3rd = "Destination_Shape"
        data_row_col_4th = "vCPUs"
        data_row_col_5th = "Memory_GB"
        data_row_col_6th = "Description"

        data_value_col = "Quantity"
        data_value_2nd_col = "Source_Cost"
        data_value_3rd_col = "Infra_Cost"
        data_value_4th_col = "OS_Licenses_Cost"
        data_value_5th_col = "GCP_Cost"

        filter_column = "CONTAINS"
    elif data_source_type == "SHEETS":
        data_source_id = [data_source["mapped"]["worksheet_id"].id, data_source["mapped"]["csv_header_length"],
                          data_source["mapped"]["csv_num_rows"]]
        data_row_col = 7  # Data, Column H, Region
        data_row_col_2nd = 8  # Data, Column I, Source_Shape
        data_row_col_3rd = 10  # Data, Column K, Destination_Shape
        data_row_col_4th = 14  # Data, Column O, vCPUs
        data_row_col_5th = 15  # Data, Column P, Memory_GB
        data_row_col_6th = 6  # Data, Column G, Description
        data_value_col = 17  # Data, Column R, Quantity
        data_value_2nd_col = 19  # Data, Column T, Source_Cost
        data_value_3rd_col = 21  # Data, Column V, Infra_Cost
        data_value_4th_col = 22  # Data, Column X, OS_Licenses_Cost
        data_value_5th_col = 23  # Data, Column Y, GCP_Cost

        filter_column = "CONTAINS"

    value_name = "Usage (Hourly)"
    value_name_2nd = "AWS Cost"
    value_name_3rd = "Machine Type Cost"
    value_name_4th = "OS Licenses Cost"
    value_name_5th = "Total GCP Cost"

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     mt_overview_worksheet_id,
                                     pivot_table_location, "SUM", None, data_row_col_2nd, None, value_name,
                                     data_value_2nd_col,
                                     value_name_2nd, filter_column, False, data_row_col_3rd, None, data_row_col_4th,
                                     None, data_row_col_5th, None, data_value_3rd_col,
                                     value_name_3rd, data_value_4th_col, value_name_4th, data_value_5th_col,
                                     value_name_5th, data_row_col_6th, None
                                     ),

    )

    # Refresh all BQ Data sources (removes 'Apply' button from pivot tables)
    res = spreadsheet.batch_update(refresh_data_sources_body)

    # Add Piechart for GCP Cost Breakdown
    chart_title = "GCP Migration Breakdown"
    piechart_row_col = 3  # Column D
    piechart_value_col = 5  # Column F
    position_data = [
        10,  # Column K
        0  # Row 21
    ]

    res = spreadsheet.batch_update(
        generate_pie_table_request(exec_overview_worksheet_id, chart_title, piechart_row_col, piechart_value_col,
                                   position_data))

    # Add Piechart for GCP Services
    chart_title = "GCP Services Breakdown"
    piechart_row_col = 0  # Column D
    piechart_value_col = 2  # Column F
    position_data = [
        14,  # Column O
        0  # Row 1
    ]

    res = spreadsheet.batch_update(
        generate_pie_table_request(gcp_overview_worksheet_id, chart_title, piechart_row_col, piechart_value_col,
                                   position_data))

    # Add Piechart for GCP Services
    chart_title = "GCP Regions Breakdown"
    piechart_row_col = 6  # Column G
    piechart_value_col = 8  # Column I
    position_data = [
        14,  # Column 0
        21  # Row 21
    ]

    res = spreadsheet.batch_update(
        generate_pie_table_request(gcp_overview_worksheet_id, chart_title, piechart_row_col, piechart_value_col,
                                   position_data))

    # Add Piechart for Instances
    chart_title = "GCP Instance Breakdown"
    piechart_row_col = 11  # Column L
    piechart_value_col = 12  # Column M
    position_data = [
        14,  # Column 0
        42  # Row 20
    ]

    res = spreadsheet.batch_update(
        generate_pie_table_request(gcp_overview_worksheet_id, chart_title, piechart_row_col, piechart_value_col,
                                   position_data))

    exec_overview_formats = [
        {
            "range": "A1:A7",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "B2",
            "format": {
                "textFormat": {
                    "italic": True,
                },
            },
        },
        {
            "range": "G1:M1",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "G",
            "format": {
                "numberFormat":
                    {
                        "type": "PERCENT",
                        "pattern": "0.0000%"
                    },
            },
        },
        {
            "range": "H",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "I",
            "format": {
                "numberFormat":
                    {
                        "type": "PERCENT",
                        "pattern": "0.0000%"
                    },
            },
        },
        {
            "range": "B",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "E:F",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "B7",
            "format": {
                "numberFormat":
                    {
                        "type": "PERCENT",
                        "pattern": "0.0000%"
                    },
            },
        },
    ]

    exec_overview_worksheet.batch_format(exec_overview_formats)

    gcp_overview_worksheet_formats = [
        {
            "range": "D1:E1",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "B:D",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "E",
            "format": {
                "numberFormat":
                    {
                        "type": "PERCENT",
                        "pattern": "0.0000%"
                    },
            },
        },
        {
            "range": "I",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        }, {
            "range": "M",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
    ]

    gcp_overview_worksheet.batch_format(gcp_overview_worksheet_formats)

    unmapped_worksheet_formats = [
        {
            "range": "B",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "F",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
    ]

    unmapped_worksheet.batch_format(unmapped_worksheet_formats)

    gcp_discounts_worksheet_formats = [
        {
            "range": "A1",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "F2:I2",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "D:F",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "G",
            "format": {
                "numberFormat":
                    {
                        "type": "PERCENT",
                        "pattern": "0.0000%"
                    },
            },
        },
        {
            "range": "H",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "L",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "K2:K4",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "N2",
            "format": {
                "textFormat": {
                    "italic": True,
                },
            },
        }
    ]

    gcp_discounts_worksheet.batch_format(gcp_discounts_worksheet_formats)

    mt_worksheet_formats = [
        {
            "range": "L1:M1",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "H:L",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "M",
            "format": {
                "numberFormat":
                    {
                        "type": "PERCENT",
                        "pattern": "0.0000%"
                    },
            },
        },
    ]

    mt_overview_worksheet.batch_format(mt_worksheet_formats)

    # Set up conditional rules (Red/Green) to GCP Overview Differences
    apply_conditional_color_rule(gcp_overview_worksheet, "D2:E", "NUMBER_GREATER", "0", [1, 0, 0])
    apply_conditional_color_rule(gcp_overview_worksheet, "D2:E", "NUMBER_LESS", "0", [0, 75, 0])

    # Set up conditional rules (Red/Green) to Exec Overview Differences
    apply_conditional_color_rule(exec_overview_worksheet, "B6", "NUMBER_GREATER", "0", [1, 0, 0])
    apply_conditional_color_rule(exec_overview_worksheet, "B6", "NUMBER_LESS", "0", [0, 75, 0])

    # Set up conditional rules (Red/Green) to Exec Overview Differences
    apply_conditional_color_rule(exec_overview_worksheet, "H:I", "NUMBER_GREATER", "0", [1, 0, 0])
    apply_conditional_color_rule(exec_overview_worksheet, "H:I", "NUMBER_LESS", "0", [0, 75, 0])

    # Set up conditional rules (Red/Green) to MT Overview Differences
    apply_conditional_color_rule(mt_overview_worksheet, "L:M", "NUMBER_GREATER", "0", [1, 0, 0])
    apply_conditional_color_rule(mt_overview_worksheet, "L:M", "NUMBER_LESS", "0", [0, 75, 0])

    # Autosize first cols in Overview worksheet
    first_col = 0
    last_col = 30
    res = spreadsheet.batch_update(autosize_worksheet(gcp_overview_worksheet_id, first_col, last_col))

    # Autosize first cols in Unmapped worksheet
    res = spreadsheet.batch_update(autosize_worksheet(unmapped_worksheet_id, first_col, last_col))

    # Autosize first cols in Exec Overview worksheet
    res = spreadsheet.batch_update(autosize_worksheet(exec_overview_worksheet_id, first_col, last_col))

    # Delete default worksheet
    worksheet = spreadsheet.worksheet("Sheet1")
    spreadsheet.del_worksheet(worksheet)

    # Reorder Worksheets
    spreadsheet.reorder_worksheets(
        [exec_overview_worksheet, gcp_overview_worksheet, unmapped_worksheet, gcp_discounts_worksheet,
         mt_overview_worksheet])  #, storage_overview_worksheet, db_overview_worksheet])

    # Get AWS Unmapped Totals
    from gspread.utils import ValueRenderOption
    aws_total_spend_array = gcp_overview_worksheet.get("B3", value_render_option=ValueRenderOption.unformatted)
    aws_total_spend = aws_total_spend_array[0]

    # Add Cost sums to AWS Unmapped Worksheet. Filter on AWS Cost column being greater than 0.
    if data_source_type == "BQ":
        data_source_id = [data_source[1]]
        data_row_col = "lineItem_ProductCode"
        data_value_col = "lineItem_UnblendedCost"
        if aws_total_spend[0] == 0:
            filter_col = "lineItem_ProductCode"
        else:
            filter_col = None
    elif data_source_type == "SHEETS":
        data_source_id = [data_source["unmapped"]["worksheet_id"].id, data_source["unmapped"]["csv_header_length"],
                          data_source["unmapped"]["csv_num_rows"]]
        data_row_col = 3  # Unmapped, Column D, lineItem_ProductCode
        data_value_col = 11  # Unmapped, Column L, lineItem_UnblendedCost
        if aws_total_spend[0] == 0:
            filter_col = "lineItem_ProductCode"
        else:
            filter_col = None

    pivot_table_location = [
        0,  # Column D
        0  # Row 1
    ]

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     unmapped_worksheet_id,
                                     pivot_table_location, "SUM", None, None, None, "AWS Cost", None, None, filter_col,
                                     False,
                                     None, None, None, None, None, None, None, None, None, None, None, None, None, None
                                     ))

    # Add Instance Region Usage Breakdown.
    if data_source_type == "BQ":
        data_source_id = [data_source[1]]
        data_row_col = "lineItem_ProductCode"
        data_value_col = "lineItem_UnblendedCost"
        data_row_col_2nd = "lineItem_UsageType"

        if aws_total_spend[0] == 0:
            filter_col = "lineItem_ProductCode"
        else:
            filter_col = None

    elif data_source_type == "SHEETS":
        data_source_id = [data_source["unmapped"]["worksheet_id"].id, data_source["unmapped"]["csv_header_length"],
                          data_source["unmapped"]["csv_num_rows"]]
        data_row_col = 3  # Unmapped, Column D, lineItem_ProductCode
        data_value_col = 11  # Unmapped, Column L, lineItem_UnblendedCost
        data_row_col_2nd = 5  # Unmapped, Column F, lineItem_UsageType

        if aws_total_spend[0] == 0:
            filter_col = "lineItem_ProductCode"
        else:
            filter_col = None

    pivot_table_location = [
        3,  # Column D
        0  # Row 1
    ]

    response = spreadsheet.batch_update(
        generate_pivot_table_request(data_source_type, data_source_id, data_row_col, data_value_col,
                                     unmapped_worksheet_id,
                                     pivot_table_location, "SUM", None, data_row_col_2nd, None, "AWS Cost", None, None,
                                     filter_col, False, None, None, None, None, None, None, None, None, None, None, None,
                                     None, None, None
                                     ))

    # Add Piechart for AWS Unmapped Services
    chart_title = "AWS Unmapped Services Breakdown"
    piechart_row_col = 0
    piechart_value_col = 1
    position_data = [
        7,  # Column H
        0  # Row 1
    ]

    if aws_total_spend[0] > 0:
        res = spreadsheet.batch_update(
            generate_pie_table_request(unmapped_worksheet_id, chart_title, piechart_row_col, piechart_value_col,
                                       position_data))

    # Re-Refresh all BQ Data sources (removes 'Apply' button from pivot tables)
    res = spreadsheet.batch_update(refresh_data_sources_body)


def generate_bq_cur_sheets(spreadsheet, worksheet_names, data_source_ids):
    overview_worksheets_name = "AWS Overview"
    details_worksheets_name = "AWS Details"
    overview_row_col_name = "lineItem_ProductCode"
    overview_value_col_name = "lineItem_UnblendedCost"

    # Create Overview Worksheet in Sheets
    overview_worksheet = spreadsheet.add_worksheet(overview_worksheets_name, 60, 25)
    details_worksheet = spreadsheet.add_worksheet(details_worksheets_name, 60, 25)

    overview_worksheet_id = overview_worksheet._properties['sheetId']
    details_worksheet_id = details_worksheet._properties['sheetId']

    source = "BQ"
    data_source = [data_source_ids[0]]

    pivot_table_location = [
        2,  # Column C
        0  # Row 1
    ]

    # AWS Services Cost
    response = spreadsheet.batch_update(
        generate_pivot_table_request(source, data_source, overview_row_col_name, overview_value_col_name,
                                     overview_worksheet_id,
                                     pivot_table_location, "SUM", None, None, None, None, None, None, None, False, None,
                                     None, None, None, None, None, None, None, None, None, None, None, None, None
                                     ))

    pivot_table_location = [
        5,  # Column F
        0  # Row 1
    ]

    overview_row_col_name = "product_region"
    overview_value_col_name = "lineItem_UnblendedCost"
    # AWS Region Cost
    response = spreadsheet.batch_update(
        generate_pivot_table_request(source, data_source, overview_row_col_name, overview_value_col_name,
                                     overview_worksheet_id,
                                     pivot_table_location, "SUM", None, None, None, None, None, None, None, False, None,
                                     None, None, None, None, None, None, None, None, None, None, None, None, None
                                     ))

    pivot_table_location = [
        8,  # Column F
        0  # Row 1
    ]

    overview_row_col_name = "product_instanceType"
    overview_value_col_name = "lineItem_UnblendedCost"
    # AWS Instance Cost
    response = spreadsheet.batch_update(
        generate_pivot_table_request(source, data_source, overview_row_col_name, overview_value_col_name,
                                     overview_worksheet_id,
                                     pivot_table_location, "SUM", None, None, None, None, None, None,
                                     "product_instanceType", False, None, None, None, None, None, None, None, None,
                                     None, None, None, None, None, None
                                     ))

    pivot_table_location = [
        0,  # Column A
        0  # Row 1
    ]

    details_row_col_name = "lineItem_ProductCode"
    details_value_col_name = "lineItem_UnblendedCost"
    # AWS Services Details Cost
    response = spreadsheet.batch_update(
        generate_pivot_table_request(source, data_source, details_row_col_name, details_value_col_name,
                                     details_worksheet_id,
                                     pivot_table_location, "SUM", None, "lineItem_UsageType", None, None, None, None,
                                     None, False, None, None, None, None, None, None, None, None, None, None, None,
                                     None, None, None
                                     ))

    pivot_table_location = [
        4,  # Column E
        0  # Row 1
    ]

    details_row_col_name = "product_region"
    details_value_col_name = "lineItem_UnblendedCost"
    # AWS Regions Details Cost
    response = spreadsheet.batch_update(
        generate_pivot_table_request(source, data_source, details_row_col_name, details_value_col_name,
                                     details_worksheet_id,
                                     pivot_table_location, "SUM", None, "product_instanceType", None, None, None, None,
                                     None, False, None, None, None, None, None, None, None, None, None, None, None,
                                     None, None, None
                                     ))

    # Add Piechart for AWS Services
    chart_title = "AWS Services Breakdown"

    position_data = [
        11,  # Column J
        0  # Row 1
    ]

    res = spreadsheet.batch_update(generate_pie_table_request(overview_worksheet_id, chart_title, 2, 3, position_data))

    # Add Piechart for AWS Regions
    chart_title = "AWS Regions Breakdown"

    position_data = [
        11,  # Column J
        21  # Row 21
    ]

    res = spreadsheet.batch_update(generate_pie_table_request(overview_worksheet_id, chart_title, 5, 6, position_data))

    # Add Piechart for AWS Instances
    chart_title = "AWS Instance Breakdown"

    position_data = [
        11,  # Column J
        42  # Row 42
    ]

    res = spreadsheet.batch_update(generate_pie_table_request(overview_worksheet_id, chart_title, 8, 9, position_data))

    # Add Piechart for AWS Detailed Instances
    chart_title = "AWS Region Instance Breakdown"

    position_data = [
        8,  # Column I
        0  # Row 1
    ]

    res = spreadsheet.batch_update(generate_pie_table_request(details_worksheet_id, chart_title, 4, 6, position_data))

    overview_worksheet.batch_update([{
        'range': "A1",
        'values': [["AWS Total Cost"]],
    }, {
        'range': "A2",
        'values': [["=SUM(D2:D)"]],
    }]
        , value_input_option="USER_ENTERED"
    )

    overview_worksheet_formats = [
        {
            "range": "A1",
            "format": {
                "textFormat": {
                    "bold": True,
                },
            },
        },
        {
            "range": "A2",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "D",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "G",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        }, {
            "range": "J",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
    ]

    overview_worksheet.batch_format(overview_worksheet_formats)

    details_worksheet_formats = [
        {
            "range": "C",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
        {
            "range": "G",
            "format": {
                "numberFormat":
                    {
                        "type": "CURRENCY"
                    },
            },
        },
    ]

    details_worksheet.batch_format(details_worksheet_formats)

    # Autosize first cols in Overview worksheet
    first_col = 0
    last_col = 30
    res = spreadsheet.batch_update(autosize_worksheet(overview_worksheet_id, first_col, last_col))

    # Refresh all BQ Data sources (removes 'Apply' button from pivot tables)
    res = spreadsheet.batch_update(refresh_data_sources_body)

    # Delete default worksheet
    worksheet = spreadsheet.worksheet("Sheet1")
    spreadsheet.del_worksheet(worksheet)

    overview_worksheet = spreadsheet.worksheet("AWS Overview")
    details_worksheet = spreadsheet.worksheet("AWS Details")

    spreadsheet.reorder_worksheets([overview_worksheet, details_worksheet])


# Import mc data from provided reports directory
def import_mc_data_sheets(mc_reports_directory, spreadsheet, credentials):
    sheets_id = spreadsheet.id
    mc_data = {}
    # Grabbing a list of files from the provided mc directory
    try:
        mc_file_list = os.listdir(mc_reports_directory)
        print("Importing MC pricing report data: ")
    except:
        print("Unable to access directory: " + mc_reports_directory)
        exit()

    client = gspread.authorize(credentials)
    sh = client.open_by_key(sheets_id)

    if len(mc_file_list) == 0:
        print(f"No files in directory {mc_reports_directory}! Exiting.")
        exit()

    # Importing all CSV files into a dictionary of dataframes
    for file in mc_file_list:
        if file.endswith(".csv"):
            file_fullpath = (mc_reports_directory + "/" + file)
            file_name, _ = file.rsplit(".csv")
            try:
                sheet_name = mc_names[file_name]
            except:
                print(f"{file_name} does not exist in config! Exiting.")
                exit()

            try:
                mc_data[file_name] = pd.read_csv(file_fullpath, low_memory=False)
            except:
                print(f"Unable to open {file}! Exiting.")
                exit()

            # Import Panda/CSV data into worksheet
            print(f"\t{file}...")
            # print(list(csv.reader(open(file_fullpath))))
            worksheet = sh.add_worksheet(title=sheet_name, rows=100, cols=30)
            sh.values_update(
                sheet_name,
                params={'valueInputOption': 'USER_ENTERED'},
                body={'values': list(csv.reader(open(file_fullpath)))})

            response = sh.batch_update(generate_protect_sheet_request(worksheet._properties['sheetId']))

    data_source = {
        "mapped": {
            "worksheet_id": sh.worksheet(mc_names["mapped"]),
            "csv_header_length": 25 + 1,
            "csv_num_rows": len(mc_data["mapped"]) + 1
        },
        "unmapped": {
            "worksheet_id": sh.worksheet(mc_names["unmapped"]),
            "csv_header_length": 25 + 1,
            "csv_num_rows": len(mc_data["unmapped"]) + 1
        },

    }

    return data_source


def google_auth(service_account_key, scope):
    # Use provided Google Service Account Key, otherwise try to use gcloud auth key to authenticate
    if service_account_key != "":
        try:
            credentials = ServiceAccountCredentials.from_json_keyfile_name(service_account_key, scope)
        except IOError:
            print("Google Service account key: " + service_account_key + " does not appear to exist! Exiting...")
            exit()
    else:
        try:
            credentials, _ = google.auth.default(scopes=scope)
        except:
            print("Unable to auth against Google...")
            exit()
    return credentials


def import_mc_into_bq(mc_reports_directory, gcp_project_id, bq_dataset_name, bq_table_prefix, service_account_key,
                      customer_name):
    # GCP Scope for auth
    scope = [
        "https://www.googleapis.com/auth/drive",
        "https://www.googleapis.com/auth/cloud-platform",
    ]

    # Google Auth
    credentials = google_auth(service_account_key, scope)
    client = gspread.authorize(credentials)

    mc_data = {}
    mc_file_list = []
    # Grabbing a list of files from the provided mc directory
    try:
        print("Importing pricing report files...")
        for file in settings_file["mc_names"].keys():
            if os.path.isfile(f"{mc_reports_directory}{file}.csv"):
                mc_file_list.append(f"{file}")
        # mc_file_list = os.listdir(f"{mc_reports_directory}/*.csv")
    except:
        print("Unable to access directory: " + mc_reports_directory)
        exit()

    # Verify MC files exist
    if len(mc_file_list) < len(settings_file["mc_names"].keys()):
        print("Required MC data files do not exist! Exiting!")
        exit()

    # Create BQ dataset
    client = bigquery.Client()
    dataset_id = f"{gcp_project_id}.{bq_dataset_name}"

    # Construct a full Dataset object to send to the API.
    dataset = bigquery.Dataset(dataset_id)

    try:
        client.get_dataset(dataset_id)  # Check if dataset exists
        print(f"Dataset {dataset_id} already exists.")
    except:
        dataset.location = "US"
        try:
            dataset = client.create_dataset(dataset, timeout=30)  # Make an API request.
        except:
            print(f"Unable to create dataset: {dataset_id}")
            exit()

        print(f"Dataset {dataset_id} created.")

    # Importing all CSV files into a dictionary of dataframes
    for file in mc_file_list:
        with open(f"{mc_reports_directory}{file}.csv", "rb") as f:
            num_lines = sum(1 for _ in f)
        if num_lines > 1:
            bq_table_name = (f"{bq_table_prefix}{file.replace('.csv', '')}")
            table_id = (f"{gcp_project_id}.{bq_dataset_name}.{bq_table_name}")
            print(f"Importing {file}.csv into BQ Table: {table_id}")
            set_gcp_project = f"gcloud config set project {gcp_project_id} >/dev/null 2>&1"

            schema = ""
            for column in mc_column_names[file].keys():
                schema = schema + f"\"{column}\":{mc_column_names[file][column]},"

            # Remove last comma
            schema = schema[:-1]
            try:
                os.system(set_gcp_project)
            except Exception as e:
                print(f"error: {e}")

            # if file.endswith(".csv"):
            file_fullpath = (f"{mc_reports_directory}{file}.csv")

            sheet_name = mc_names[file]
            mc_data[file] = pd.read_csv(file_fullpath, low_memory=False)
            # Replacing column names since BQ doesn't like them with () & the python library "column character map" version doesn't appear to work.

            # Ensure the various MC & CUR import versions have the same column names
            if file == 'mapped':
                mc_data[file].rename(columns={
                    "Memory (GB)": "Memory_GB",
                    "External Memory (GB)": "External_Memory_GB",
                    "Sub-Type 1": "Sub_Type_1",
                    "Sub-Type 2": "Sub_Type_2",
                    "Dest Series": "Destination_Series",
                    "Extended Memory GB": "External_Memory_GB",
                    "Dest Shape": "Destination_Shape",
                    "OS or Licenses Cost": "OS_Licenses_Cost",
                    "Dest. Shape": "Destination_Shape",
                    "Dest. Series": "Destination_Series",
                    "OS / Licenses Cost": "OS_Licenses_Cost",
                    "Account/Subscription": "Account_Or_Subscription",
                    "Ext. Memory (GB)": "External_Memory_GB"
                }, inplace=True)

            if file == 'unmapped':
                mc_data[file].rename(columns={
                    "ID": "identity_LineItemIds"
                }, inplace=True)

            if file == 'credit-and-refund':
                mc_data[file].rename(columns={
                    "ID": "identity_LineItemIds"
                }, inplace=True)

            # Ensure no spaces exist in any column names
            mc_data[file].rename(columns=lambda x: x.replace(" ", "_"), inplace=True)

            # More ensuring the various MC & CUR import versions have the same column names
            mc_data[file].rename(columns=lambda x: x.replace("product_", "lineItem_"), inplace=True)

            schema = []
            # Create Schema Fields for BQ
            for column in mc_column_names[file].keys():
                if mc_column_names[file][column] == 'STRING':
                    schema.append(bigquery.SchemaField(column, bigquery.enums.SqlTypeNames.STRING))
                elif mc_column_names[file][column] == 'FLOAT64':
                    schema.append(bigquery.SchemaField(column, bigquery.enums.SqlTypeNames.FLOAT64))

                # col_count += 1

            job_config = bigquery.LoadJobConfig(

                autodetect=True,
                skip_leading_rows=1,
                write_disposition=bigquery.WriteDisposition.WRITE_TRUNCATE,
                create_disposition=bigquery.CreateDisposition.CREATE_IF_NEEDED,
                column_name_character_map="V2",
                allow_quoted_newlines=True,
                schema=schema,
                source_format=bigquery.SourceFormat.CSV
            )

            job = client.load_table_from_dataframe(
                mc_data[file], table_id, job_config=job_config
            )  # Make an API request.
            job.result()  # Wait for the job to complete.

            mc_data[file] = mc_data[file].iloc[0:0]

            table = client.get_table(table_id)  # Make an API request.
            print(
                "Loaded {} rows and {} columns to {}".format(
                    table.num_rows, len(table.schema), table_id
                )
            )
        else:
            print(f"Skipping {file}.csv since there is no Migration Center data in the file.")

    print("Completed loading of Migration Center Data into Big Query.")


def import_cur_into_bq(mc_reports_directory, gcp_project_id, bq_dataset_name, bq_table, service_account_key,
                       customer_name):
    # GCP Scope for auth
    scope = [
        "https://www.googleapis.com/auth/drive",
        "https://www.googleapis.com/auth/cloud-platform",
    ]

    #Google Auth
    credentials = google_auth(service_account_key, scope)
    client = gspread.authorize(credentials)

    cur_data = {}
    cur_file_list = [f for f in os.listdir(mc_reports_directory) if
                     os.path.isfile(os.path.join(mc_reports_directory, f))]

    # Create BQ dataset
    client = bigquery.Client()
    dataset_id = f"{gcp_project_id}.{bq_dataset_name}"

    # Construct a full Dataset object to send to the API.
    dataset = bigquery.Dataset(dataset_id)

    try:
        client.get_dataset(dataset_id)  # Check if dataset exists
        print(f"Dataset {dataset_id} already exists.")
    except:
        dataset.location = "US"
        try:
            dataset = client.create_dataset(dataset, timeout=30)  # Make an API request.
        except:
            print(f"Unable to create dataset: {dataset_id}")
            exit()

        print(f"Dataset {dataset_id} created.")

    table_id = (f"{gcp_project_id}.{bq_dataset_name}.{bq_table}")
    # Deleting table first if exists

    client.delete_table(table_id, not_found_ok=True)

    # Importing all CSV files into a dictionary of dataframes
    for file in cur_file_list:
        with open(f"{mc_reports_directory}{file}", "rb") as f:
            num_lines = sum(1 for _ in f)
        if num_lines > 1:
            print(f"Importing {file} into BQ Table: {table_id}")
            set_gcp_project = f"gcloud config set project {gcp_project_id} >/dev/null 2>&1"

            # schema = ""
            # for column in mc_column_names[file].keys():
            #     schema = schema + f"\"{column}\":{mc_column_names[file][column]},"
            #
            # # Remove last comma
            # schema = schema[:-1]
            try:
                os.system(set_gcp_project)
            except Exception as e:
                print(f"error: {e}")

            # if file.endswith(".csv"):
            file_fullpath = (f"{mc_reports_directory}{file}")

            cur_data[file] = pd.read_csv(file_fullpath, low_memory=False)

            # Ensure no spaces exist in any column names
            cur_data[file].rename(columns=lambda x: x.replace(" ", "_"), inplace=True)
            cur_data[file].rename(columns=lambda x: x.replace("/", "_"), inplace=True)

            job_config = bigquery.LoadJobConfig(

                autodetect=True,
                skip_leading_rows=1,
                write_disposition=bigquery.WriteDisposition.WRITE_APPEND,
                create_disposition=bigquery.CreateDisposition.CREATE_IF_NEEDED,
                column_name_character_map="V2",
                allow_quoted_newlines=True,
                #schema=schema,
                source_format=bigquery.SourceFormat.CSV
            )

            job = client.load_table_from_dataframe(
                cur_data[file], table_id, job_config=job_config
            )  # Make an API request.
            job.result()  # Wait for the job to complete.

            cur_data[file] = cur_data[file].iloc[0:0]

            table = client.get_table(table_id)  # Make an API request.
            print(
                "Loaded {} rows and {} columns to {}".format(
                    table.num_rows, len(table.schema), table_id
                )
            )
        else:
            print(f"Skipping {file} since there is no data in the file.")

    print("Completed loading of AWS CUR Data into Big Query.\n")


def create_looker_url(looker_template, customer_name, datetime, gcp_project_id, bq_dataset_name, bq_table):
    # Looker Settings
    looker_url_prefix = "https://lookerstudio.google.com/reporting/create?c.reportId="
    looker_report_name = f"AWS -> GCP Pricing Analysis: {customer_name}, {datetime}"
    looker_report_name = urllib.parse.quote_plus(looker_report_name)

    if looker_template == 'MC':
        looker_template_id = default_mc_looker_template_id
        ds0_bq_datasource_name = "mapped"
        ds0_bq_table = f"{bq_table}mapped"

        ds1_bq_datasource_name = "unmapped"
        ds1_bq_table = f"{bq_table}unmapped"

        looker_report_url = f"{looker_url_prefix}{looker_template_id}&r.reportName={looker_report_name}&ds.ds0.connector=bigQuery&ds.ds0.datasourceName={ds0_bq_datasource_name}&ds.ds0.projectId={gcp_project_id}&ds.ds0.type=TABLE&ds.ds0.datasetId={bq_dataset_name}&ds.ds0.tableId={ds0_bq_table}&ds.ds1.connector=bigQuery&ds.ds1.datasourceName={ds1_bq_datasource_name}&ds.ds1.projectId={gcp_project_id}&ds.ds1.type=TABLE&ds.ds1.datasetId={bq_dataset_name}&ds.ds1.tableId={ds1_bq_table}"

    if looker_template == 'CUR':
        looker_template_id = default_cur_looker_template_id
        ds0_bq_datasource_name = "cur"
        ds0_bq_table = f"{bq_table}"

        looker_report_url = f"{looker_url_prefix}{looker_template_id}&r.reportName={looker_report_name}&ds.ds0.connector=bigQuery&ds.ds0.datasourceName={ds0_bq_datasource_name}&ds.ds0.projectId={gcp_project_id}&ds.ds0.type=TABLE&ds.ds0.datasetId={bq_dataset_name}&ds.ds0.tableId={ds0_bq_table}"

    return looker_report_url


# Parse CLI Arguments
def parse_cli_args():
    parser = argparse.ArgumentParser(prog='google-mc-c2c-data-import.py',
                                     usage='%(prog)s -d <mc report directory>\nThis creates an instance mapping between cloud providers and GCP')
    parser.add_argument('-d', metavar='Data Directory',
                        help='Directory containing MC report output or AWS CUR data.',
                        required=True, )
    parser.add_argument('-c', metavar='Customer Name', help='Customer Name',
                        required=False, )
    parser.add_argument('-e', metavar='Email Addresses', help='Emails to share Google Sheets with (comma separated)',
                        required=False, )
    parser.add_argument('-s', metavar='Google Sheets ID', required=False,
                        help='Use existing Google Sheets instead of creating a new one. Takes Sheets ID')
    parser.add_argument('-k', metavar='SA JSON Keyfile', required=False,
                        help='Google Service Account JSON Key File. Both Drive & Sheets API in GCP Project must be enabled! ')
    parser.add_argument('-b', action='store_true', required=False,
                        help='Import Migration Center data files into Biq Query Dataset.\nGCP BQ API must be enabled! ')
    parser.add_argument('-a', action='store_true', required=False,
                        help='Import AWS CUR file into Biq Query Dataset.\nGCP BQ API must be enabled! ')
    parser.add_argument('-l', action='store_true', required=False,
                        help='Display Looker Report URL. Migration Center or AWS CUR BQ Import must be enabled! ')
    parser.add_argument('-r', metavar='Looker Templ ID', required=False,
                        help='Replaces Default Looker Report Template ID')
    parser.add_argument('-n', action='store_true', required=False,
                        help='Create a Google Connected Sheets to newly created Big Query')
    parser.add_argument('-o', action='store_true', required=False,
                        help='Do not import to BQ, use an existing BQ instance (-i) and only create connected Sheets & Looker artifacts.')
    parser.add_argument('-i', metavar='BQ Connect Info', required=False,
                        help='BQ Connection Info: Format is <GCP Project ID>.<BQ Dataset Name>.<BQ Table Prefix>, i.e. googleproject.bqdataset.bqtable_prefix')
    return parser.parse_args()


def main():
    args = parse_cli_args()

    enable_cur_import = args.a
    enable_bq_import = args.b
    mc_reports_directory = args.d
    connect_sheets_bq = args.n
    sheets_emails = args.e
    do_not_import_data = args.o
    bq_connection_info = args.i

    if args.r is not None:
        looker_template_id = args.r
    else:
        if args.b is True:
            looker_template_id = default_mc_looker_template_id
        if args.a is True:
            looker_template_id = default_cur_looker_template_id

    if args.l is True:
        display_looker = "Yes"
    else:
        display_looker = "No"

    print(f"Migration Center C2C Data Import, {version}")

    if args.c is not None:
        customer_name = args.c
    else:
        customer_name = "No Name Customer, Inc."

    print("Customer: " + customer_name)

    if mc_reports_directory is not None:
        print("Migration Center Reports directory: " + mc_reports_directory)
    else:
        print("Migration Center Reports directory not defined, exiting!")
        exit()

    if connect_sheets_bq is True and (
            enable_bq_import is False and enable_cur_import is False and do_not_import_data is False):
        print("Must enable Big Query with -b or -a before creating a Connected BQ Google Sheets!")
        exit()

    if enable_bq_import is not True and enable_cur_import is not True and do_not_import_data is not True:

        check_csv_size(mc_reports_directory)

        if sheets_emails is not None:
            sheets_email_addresses = sheets_emails.split(",")
            print("Sharing Sheets with: ")
            for email in sheets_email_addresses:
                print(email)
        else:
            sheets_email_addresses = ""

        if args.k is not None:
            service_account_key = args.k
            print("Using Google Service Account key: " + service_account_key)
        else:
            service_account_key = ""

        if args.s is not None:
            sheets_id = args.s
        else:
            sheets_id = ""

        spreadsheet, credentials = create_google_sheets(customer_name, sheets_email_addresses, service_account_key,
                                                        sheets_id)

        # import_mc_data_old(mc_reports_directory, spreadsheet, credentials)
        # worksheet_names = [mc_names["mapped"], mc_names["unmapped"]]
        worksheet_names = []
        data_source = import_mc_data_sheets(mc_reports_directory, spreadsheet, credentials)
        generate_mc_sheets(spreadsheet, worksheet_names, "SHEETS", data_source, mc_names["unmapped"])

        spreadsheet_url = 'https://docs.google.com/spreadsheets/d/%s' % spreadsheet.id

        print("Migration Center Pricing Report for " + customer_name + ": " + spreadsheet_url)
    else:
        if bq_connection_info is None:
            print("No Big Query connection information provided. Exiting!")
            exit()

        bq_connection_info = bq_connection_info

        (gcp_project_id, bq_dataset_name, bq_table_prefix) = bq_connection_info.split(".")

        bq_tables = []

        if enable_cur_import is True:
            print(f"BQ Table: {bq_table_prefix}")
            bq_tables.append(bq_table_prefix)
            overview_worksheets_name = "AWS Overview"
        else:
            print(f"BQ Table Prefix: {bq_table_prefix}")
            for table in list(mc_names.keys()):
                bq_tables.append(f'{bq_table_prefix}{table}')

        if do_not_import_data is False:
            print("Importing data into Big Query...")
            print(f"GCP Project ID: {gcp_project_id}")
            print(f"BQ Dataset Name: {bq_dataset_name}")

            if args.k is not None:
                service_account_key = args.k
                print("Using Google Service Account key: " + service_account_key)
            else:
                service_account_key = ""

            if args.c is not None:
                customer_name = args.c
            else:
                customer_name = "No Name Customer, Inc."

            if enable_bq_import is True and enable_cur_import is False:
                print("Migration Center Data import...")
                import_mc_into_bq(mc_reports_directory, gcp_project_id, bq_dataset_name, bq_table_prefix,
                                  service_account_key, customer_name)

            if enable_bq_import is True and enable_cur_import is True:
                print("Unable to import Migration Center & AWS CUR data at the same time. Please do each separately.")
                exit()

            if enable_cur_import is True and enable_bq_import is False:
                print("AWS CUR import...")
                import_cur_into_bq(mc_reports_directory, gcp_project_id, bq_dataset_name, bq_table_prefix,
                                   service_account_key,
                                   customer_name)

        if do_not_import_data is True:
            if enable_bq_import is not True and enable_cur_import is not True:
                print("Please specific whether to generate a Migration Center report (-b) or AWS CUR report (-a).")
                exit()

        if enable_bq_import is True and display_looker == "Yes":
            looker_report_url = create_looker_url("MC", customer_name, datetime, gcp_project_id, bq_dataset_name,
                                                  bq_table_prefix)
            print(f"\nLooker URL: {looker_report_url}\n")

        elif enable_cur_import is True and display_looker == "Yes":
            looker_report_url = create_looker_url("CUR", customer_name, datetime, gcp_project_id, bq_dataset_name,
                                                  bq_table_prefix)
            print(f"\nLooker URL: {looker_report_url}\n")

        if connect_sheets_bq is True:
            if sheets_emails is not None:
                sheets_email_addresses = sheets_emails.split(",")
                print("Sharing Sheets with: ")
                for email in sheets_email_addresses:
                    print(email)
            else:
                sheets_email_addresses = ""

            if args.k is not None:
                service_account_key = args.k
                print("Using Google Service Account key: " + service_account_key)
            else:
                service_account_key = ""

            if args.s is not None:
                sheets_id = args.s
            else:
                sheets_id = ""

            # Create New Google Sheet
            spreadsheet, credentials = create_google_sheets(customer_name, sheets_email_addresses, service_account_key,
                                                            sheets_id)

            data_source_ids = []
            worksheet_names = []
            unmapped_worksheet_name = ""

            # Connect each BG Table to a Worksheet
            for bq_table in bq_tables:
                response = spreadsheet.batch_update(connect_bq_to_sheets(gcp_project_id, bq_dataset_name, bq_table))
                bq_table_worksheet_id = spreadsheet.worksheet(bq_table)

                if do_not_import_data is True:
                    worksheet_names.append(bq_table_worksheet_id)

                # Autosize first cols in BQ table worksheet
                # res = spreadsheet.batch_update(autosize_worksheet(bq_table_worksheet_id, 0, 10))

                # Get dataource ID from batch update response
                data_source_ids.append(response['replies'][0]['addDataSource']['dataSource']['dataSourceId'])
                # print(response)
                if 'unmapped' in response['replies'][0]['addDataSource']['dataSource']['spec']['bigQuery']['tableSpec'][
                    'tableId']:
                    unmapped_worksheet_name = \
                        response['replies'][0]['addDataSource']['dataSource']['spec']['bigQuery']['tableSpec'][
                            'tableId']

            # pivot_table_location = [0, 0]
            if enable_bq_import is True:
                generate_mc_sheets(spreadsheet, worksheet_names, "BQ", data_source_ids, unmapped_worksheet_name)

            if enable_cur_import is True:
                generate_bq_cur_sheets(spreadsheet, worksheet_names, data_source_ids)

            spreadsheet_url = "https://docs.google.com/spreadsheets/d/%s" % spreadsheet.id

            print("Migration Center Sheets: " + spreadsheet_url)


if __name__ == "__main__":
    main()
