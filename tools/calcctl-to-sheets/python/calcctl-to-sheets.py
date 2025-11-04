import pandas as pd
import gspread
import csv
from oauth2client.service_account import ServiceAccountCredentials
from gspread_formatting import *
import os
import argparse
import yaml
from gspread_helper import load_spreadsheet, create_spreadsheet_from_template
import re

calcctl_names = {
    "mapped": "Mapped Data",
    "unmapped": "Unmapped Data",
    "uncharged-on-gcp": "Uncharged on GCP",
    "source-aggregated": "Original Data, Aggregated",
    "marketplace": "Marketplace"
}

def add_inferred_discounts_to_summary(reports_dir, spreadsheet):
    """Adds inferred discount information to the 'Executive Summary' worksheet."""
    summary_file_path = os.path.join(reports_dir, 'summary.yaml')

    if not os.path.exists(summary_file_path):
        print(f"Warning: Assumptions file not found at {summary_file_path}")
        return

    print("Checking for inferred discounts...")
    with open(summary_file_path, 'r') as f:
        data = yaml.safe_load(f)

    inferred_discounts = data.get('inferred_discounts')
    if not inferred_discounts or not inferred_discounts.get('could_infer'):
        print("No inferred discounts found or discounts could not be inferred.")
        return

    general_discount = inferred_discounts.get('general', 'Could not Infer')
    print(f"General Discount from source files: {general_discount}")

    try:
        worksheet = spreadsheet.worksheet("Executive Summary")

        cells_to_update = ['C13', 'C8', 'D8']
        updates_to_perform = [{'range': cell, 'values': [[general_discount]]} for cell in cells_to_update]

        average_by_product = inferred_discounts.get('by_product')
        if average_by_product:
            print("Adding inferred discounts by product...")
            updates_to_perform.append({'range': 'B15', 'values': [["Inferred Discount per Product"]]})

            product_discounts_data = list(average_by_product.items())
            if product_discounts_data:
                start_row = 16
                end_row = start_row + len(product_discounts_data) - 1
                updates_to_perform.append({'range': f'B{start_row}:C{end_row}', 'values': product_discounts_data})

        if updates_to_perform:
            worksheet.batch_update(updates_to_perform, value_input_option='USER_ENTERED')
            print("Successfully updated 'Executive Summary' with inferred discounts.")
    except gspread.exceptions.WorksheetNotFound:
        print("Warning: 'Executive Summary' worksheet not found.  Cannot add inferred discounts.")
    except Exception as e:
        print(f"An error occurred while adding inferred discounts: {e}")


def validate_reports_directory(reports_dir):
    """Checks if the report directory exists and all required report files are present."""
    if not os.path.isdir(reports_dir):
        print(f"Error: Report directory not found at '{reports_dir}'")
        print("Exiting.")
        exit(1)

    print(f"Validating contents of report directory: {reports_dir}")

    expected_files = [f"{name}.csv" for name in calcctl_names.keys()]
    expected_files.append('summary.yaml')

    missing_files = []
    for filename in expected_files:
        file_path = os.path.join(reports_dir, filename)
        if not os.path.exists(file_path):
            missing_files.append(filename)

    if missing_files:
        print(f"Error: The following required files are missing from '{reports_dir}':")
        for f in missing_files:
            print(f"- {f}")
        print("Exiting.")
        exit(1)


def load_assumptions(reports_dir, spreadsheet):
    """Loads assumptions from summary.yaml into a worksheet."""
    assumptions_file_path = os.path.join(reports_dir, 'summary.yaml')
    if not os.path.exists(assumptions_file_path):
        print(f"Warning: Assumptions file not found at {assumptions_file_path}")
        return

    print("Loading assumptions from summary.yaml...")
    with open(assumptions_file_path, 'r') as f:
        data = yaml.safe_load(f)

    assumptions = data.get('assumptions')
    if not assumptions:
        print("No 'assumptions' section in summary.yaml")
        return

    try:
        worksheet = spreadsheet.worksheet("Assumptions")
        worksheet.clear()
        print("Cleared existing 'Assumptions' worksheet.")
    except gspread.exceptions.WorksheetNotFound:
        worksheet = spreadsheet.add_worksheet(title="Assumptions", rows=200, cols=1)
        print("Created 'Assumptions' worksheet.")

    all_values = []
    formats_to_apply = []
    current_row = 1

    for section, items in assumptions.items():
        # Add section title
        section_title = section.replace('_', ' ').title()
        all_values.append([section_title])
        formats_to_apply.append({
            'range': f'A{current_row}',
            'format': {'textFormat': {'bold': True, 'fontSize': 14}}
        })
        current_row += 1

        # Add section items
        for item in items:
            all_values.append([f'• {item}'])
            current_row += 1
        
        # Add a blank row after each section
        all_values.append([''])
        current_row += 1

    if all_values:
        worksheet.update(range_name=f'A1:A{len(all_values)}', values=all_values, value_input_option='USER_ENTERED')
    
    if formats_to_apply:
        worksheet.batch_format(formats_to_apply)
    
    # Autosize column A
    worksheet.spreadsheet.batch_update({
        "requests": [{"autoResizeDimensions": {"dimensions": {"sheetId": worksheet.id, "dimension": "COLUMNS", "startIndex": 0, "endIndex": 1}}}]
    })

    print("Successfully loaded assumptions into 'Assumptions' worksheet.")


# Import calcctl data from provided reports directory
def import_calcctl_data(calcctl_reports_directory, sh):
    # Grabbing a list of files from the provided calcctl directory
    try:
        calcctl_file_list = os.listdir(calcctl_reports_directory)
        print("Importing calcctl files...")
    except OSError as e:
        print(f"Error: Unable to access directory '{calcctl_reports_directory}': {e}")
        exit(1)

    # Importing all CSV files into a dictionary of dataframes
    for file in calcctl_file_list:
        if file.endswith(".csv"):
            file_fullpath = os.path.join(calcctl_reports_directory, file)
            file_name, _ = file.rsplit(".csv")
            if file_name in calcctl_names:
                sheet_name = calcctl_names[file_name]
                with open(file_fullpath, 'r', encoding='utf-8') as f_csv:
                    csv_content = list(csv.reader(f_csv))
                sh.values_update(
                    sheet_name,
                    params={'valueInputOption': 'USER_ENTERED'},
                    body={'values': csv_content})


# Parse CLI Arguments
def parse_cli_args():
    parser = argparse.ArgumentParser(prog='calcctl-to-sheets.py',
                                     usage='%(prog)s -d <calcctl report directory> ./\nThis creates an instance mapping between cloud providers and GCP')
    parser.add_argument('-d', metavar='calcctl Reports Directory',
                        help='Directory containing calcctl report output. Contains mapped.csv, unmapped.csv, etc',
                        required=True, )
    parser.add_argument('-c', metavar='Customer Name', help='Customer Name',
                        required=False, )
    parser.add_argument('-e', metavar='Email Addresses', help='Emails to share Google Sheets with (comma separated)',
                        required=False, )
    parser.add_argument('-s', metavar='Google Sheets ID', required=False,
                        help='Use existing Google Sheets instead of creating a new one. Takes Sheets ID')
    parser.add_argument('-k', metavar='SA JSON Keyfile', required=False,
                        help='Google Service Account JSON Key File. Both Drive & Sheets API in GCP Project must be enabled! ')
    return parser.parse_args()


def main():
    args = parse_cli_args()
    reports_dir = args.d
    sheets_id = args.s
    customer_name = args.c if args.c else "No Name Customer, Inc."
    sa_key = args.k

    validate_reports_directory(reports_dir)

    sheets_email_addresses = []
    if args.e:
        sheets_email_addresses = [email.strip() for email in args.e.split(',') if email.strip()]

    if sheets_id:
        print(f"Loading existing spreadsheet: {sheets_id}")
        spreadsheet = load_spreadsheet(sheets_id, service_account_key=sa_key)
    else:
        template_spreadsheet_id = "16vkNCSZOfdKaD0pnixfdFSSqdMxAmT3WCHOL1CDGJzs"
        spreadsheet = create_spreadsheet_from_template(
            template_spreadsheet_id=template_spreadsheet_id,
            customer_name=customer_name,
            service_account_key=sa_key,
            sheets_email_addresses=sheets_email_addresses
        )

    if spreadsheet:
        print("\nSpreadsheet is ready.")
        worksheets = spreadsheet.worksheets()
        print("Worksheets:")
        for ws in worksheets:
            print(f"- {ws.title}")
        
        import_calcctl_data(reports_dir, spreadsheet)
        load_assumptions(reports_dir=reports_dir,
                         spreadsheet=spreadsheet)
        add_inferred_discounts_to_summary(reports_dir, spreadsheet)
        print("\nAll data has been loaded successfully.")

        if spreadsheet and sheets_email_addresses:
            print(f"\nSharing spreadsheet with: {', '.join(sheets_email_addresses)}")
            for email in sheets_email_addresses:
                try:
                    spreadsheet.share(email, perm_type='user', role='writer', notify=False)
                    print(f" - Shared with {email} as writer.")
                except Exception as e:
                    print(f" - Failed to share with {email}: {e}")

        print("\n")
        print(f"Spreadsheet URL: {spreadsheet.url}")

if __name__ == "__main__":
    main()