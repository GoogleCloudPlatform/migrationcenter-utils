import gspread
import google.auth
import datetime
from oauth2client.service_account import ServiceAccountCredentials


def _google_auth(service_account_key, scope):
    """Handles Google authentication using a service account key or default credentials."""
    if service_account_key:
        try:
            return ServiceAccountCredentials.from_json_keyfile_name(service_account_key, scope)
        except IOError:
            print(f"Google Service account key: {service_account_key} does not appear to exist! Exiting...")
            exit(1)
    else:
        try:
            credentials, _ = google.auth.default(scopes=scope)
            return credentials
        except Exception as e:
            print(f"Unable to auth against Google: {e}")
            print("Please run 'gcloud auth application-default login --scopes=https://www.googleapis.com/auth/drive,https://www.googleapis.com/auth/spreadsheets'")
            exit(1)


def load_spreadsheet(spreadsheet_id: str,
                     service_account_key: str = None) -> gspread.Spreadsheet | None:
    """
    Loads a Google Spreadsheet by its ID.

    Args:
        spreadsheet_id: The ID of the spreadsheet to load.
        service_account_key: Path to the service account JSON key file.
                             If None, uses default application credentials.

    Returns:
        A gspread.Spreadsheet object if successful, otherwise None.
    """
    print(f"Attempting to load spreadsheet with ID: {spreadsheet_id}")
    scope = ["https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/spreadsheets"]

    credentials = _google_auth(service_account_key, scope)

    try:
        client = gspread.authorize(credentials)
        spreadsheet = client.open_by_key(spreadsheet_id)
        print(f"Successfully loaded spreadsheet: '{spreadsheet.title}'")
        return spreadsheet
    except gspread.exceptions.SpreadsheetNotFound:
        print(f"Error: Spreadsheet with ID '{spreadsheet_id}' not found or you don't have permission to access it.")
        return None
    except Exception as e:
        print(f"An unexpected error occurred: {e}")
        return None


def create_spreadsheet_from_template(template_spreadsheet_id: str,
                                     customer_name: str,
                                     service_account_key: str = None,
                                     sheets_email_addresses: list = None) -> gspread.Spreadsheet | None:
    """
    Creates a new Google Spreadsheet by copying a template.

    Args:
        template_spreadsheet_id: The ID of the template spreadsheet to copy.
        customer_name: The name of the customer for the new spreadsheet title.
        service_account_key: Path to the service account JSON key file.
        sheets_email_addresses: A list of email addresses to share the new sheet with.

    Returns:
        A gspread.Spreadsheet object for the new spreadsheet if successful, otherwise None.
    """
    if sheets_email_addresses is None:
        sheets_email_addresses = []

    scope = ["https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/spreadsheets"]
    credentials = _google_auth(service_account_key, scope)

    try:
        client = gspread.authorize(credentials)
        now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M")
        new_sheet_title = f"calcctl Report: {customer_name} - {now}"

        print(f"Creating a new spreadsheet '{new_sheet_title}' from template...")
        new_spreadsheet = client.copy(template_spreadsheet_id, title=new_sheet_title, copy_permissions=False)

        if sheets_email_addresses:
            print(f"Sharing with: {', '.join(sheets_email_addresses)}")
            for email in sheets_email_addresses:
                new_spreadsheet.share(email, perm_type='user', role='writer', notify=False)

        print(f"Successfully created spreadsheet. URL: {new_spreadsheet.url}")
        return new_spreadsheet
    except gspread.exceptions.APIError as e:
        print(f"An API error occurred while copying the spreadsheet: {e}")
        return None
    except Exception as e:
        print(f"An unexpected error occurred: {get_error_message(e)}")
        return None
    

def get_error_message(e: Exception) -> str:
    error_message = str(e)
    if error_message == "":
        error_message = str(e.__cause__)
    return error_message

def get_worksheet_by_title(spreadsheet: gspread.Spreadsheet, title: str) -> gspread.Worksheet | None:
    """
    Finds and returns a worksheet by its title.

    Args:
        spreadsheet: The gspread.Spreadsheet object to search within.
        title: The title of the worksheet to find.

    Returns:
        A gspread.Worksheet object if found, otherwise None.
    """
    try:
        worksheet = spreadsheet.worksheet(title)
        print(f"Found worksheet with title: '{title}'")
        return worksheet
    except gspread.exceptions.WorksheetNotFound:
        print(f"Error: Worksheet with title '{title}' not found in spreadsheet '{spreadsheet.title}'.")
        return None
    except Exception as e:
        print(f"An unexpected error occurred while trying to get worksheet '{title}': {e}")
        return None

def _col_to_a1(n) -> str:
    """Converts a column number to A1 notation."""
    string = ""
    while n > 0:
        n, remainder = divmod(n - 1, 26)
        string = chr(65 + remainder) + string
    return string


def find_columns_by_header(worksheet, headers_to_find) -> dict[str, str]:
    """
    Finds column letters for given header texts in the first row of a worksheet.

    Args:
        worksheet: The gspread worksheet object.
        headers_to_find: A list of header strings to search for.

    Returns:
        A dictionary mapping header text to its alphabetical column name (e.g., 'A', 'B').
        If a header is not found, its value will be None.
    """
    print(f"\nSearching for columns in '{worksheet.title}' worksheet...")
    try:
        header_row = worksheet.row_values(1)
        found_columns = {header: None for header in headers_to_find}
        for header in headers_to_find:
            try:
                # gspread columns are 1-indexed
                col_index = header_row.index(header) + 1
                col_letter = _col_to_a1(col_index)
                found_columns[header] = col_letter
                print(f"Found '{header}' in column {col_letter}")
            except ValueError:
                print(f"Header '{header}' not found in the first row.")
        return found_columns
    except gspread.exceptions.APIError as e:
        print(f"An API error occurred while reading worksheet '{worksheet.title}': {e}")
        return {header: None for header in headers_to_find}