# Google Migration Center C2C Data Import

This application can automatically create a [Google Sheets](https://sheets.google.com/) from a Migration Center generated pricing report OR import the Migration Center Report and/or AWS CUR into [Big Query](https://cloud.google.com/bigquery). If imported into BQ, a [Looker Studio](https://lookerstudio.google.com/) report can also be created.

**NOTE** - Google Sheets has a limitation of 5 million cells and this size limit prevents the import of large (multi-gigabyte) Migration Center pricing reports. If you hit the cell limitation, consider using the -b argument to import the MC data into Big Query instead. 

Further Instuctions on using the Google Sheets Data Connector with Big Query can be found [here](https://support.google.com/docs/answer/9702507).


---
### Google Cloud SDK

In order for the code to authenticate against Google, you must have the Google Cloud SDK (gcloud) installed.

Instructions for installing the Google Cloud SDK can be found in the [Install the gcloud CLI](https://cloud.google.com/sdk/docs/install) guide.

---
### Python Environment

This application requires Python3 to be installed. Once it is installed, you can install the required Python modules using the included `requirements.txt` file:

```shell
$ cd c2c-report/python
$ pip3 install -r requirements.txt
```
#### Using with virtual environment 

If you wish to run the application inside of a python virtual environment, you can run the following:

```shell
$ sudo apt install python3.11-venv
$ cd c2c-report/python/
$ python3 -m venv ../venv
$ source ../venv/bin/activate
(venv) $ pip3 install -r requirements.txt
```

Now you can run the python script anytime by switching to the virtual environment:

```shell
$ cd c2c-report/python
$ source ../venv/bin/activate
(venv) $ python c2c-report.py ....
```

---
### Authenticate to Google

In order to use the Google Drive/Sheets API, you must have a Google project setup.
Once you have a project setup, you can run the following to authenticate against the Google project using your Google account & run the application:

```shell
$ gcloud auth login
$ gcloud config set project <PROJECT-ID>
$ gcloud services enable drive.googleapis.com sheets.googleapis.com
$ gcloud auth application-default login --scopes='https://www.googleapis.com/auth/drive','https://www.googleapis.com/auth/cloud-platform'
```

If you need to save the data to Big Query, then you can authenticate using the following commands:


```shell
$ gcloud auth login
$ gcloud config set project <PROJECT-ID>
$ gcloud services enable bigquery.googleapis.com
$ gcloud auth application-default login --scopes='https://www.googleapis.com/auth/drive','https://www.googleapis.com/auth/cloud-platform','https://www.googleapis.com/auth/bigquery'
```

---
#### Application Arguments
```shell 
$ cd c2c-report/python
$ python c2c-report.py -h
usage: c2c-report.py -d <mc report directory>
This creates an instance mapping between cloud providers and GCP

options:
  -h, --help           show this help message and exit
  -d Data Directory    Directory containing MC report output or AWS CUR data.
  -c Customer Name     Customer Name
  -e Email Addresses   Emails to share Google Sheets with (comma separated)
  -s Google Sheets ID  Use existing Google Sheets instead of creating a new one. Takes Sheets ID
  -k SA JSON Keyfile   Google Service Account JSON Key File. Both Drive & Sheets API in GCP Project must be enabled!
  -b                   Import Migration Center data files into Biq Query Dataset. GCP BQ API must be enabled!
  -a                   Import AWS CUR file into Biq Query Dataset. GCP BQ API must be enabled!
  -l                   Display Looker Report URL. Migration Center or AWS CUR BQ Import must be enabled!
  -r Looker Templ ID   Replaces Default Looker Report Template ID
  -n                   Create a Google Connected Sheets to newly created Big Query
  -o                   Do not import to BQ, use an existing BQ instance (-i) and only create connected Sheets & Looker artifacts.
  -i BQ Connect Info   BQ Connection Info: Format is <GCP Project ID>.<BQ Dataset Name>.<BQ Table Prefix>, i.e. googleproject.bqdataset.bqtable_prefix

```

---
#### Example Run: Google Sheets Creation


```shell 
$ cd c2c-report/python
$ python c2c-report.py -d ~/mc-test/ -c "Demo Customer, Inc" 
Migration Center C2C Data Import, v0.2
Customer: Demo Customer, Inc
Migration Center Reports directory: /Users/user/mc-test/
Checking CSV sizes...

Creating new Google Sheets...
Importing pricing report data...
Migration Center Pricing Report for Demo Customer, Inc: https://docs.google.com/spreadsheets/d/123456789
```

---
#### Example Run: Big Query Import with Looker Report

```shell 
$ cd c2c-report/python
$ python c2c-report.py -d ~/mc-test/ -c "Demo Customer, Inc" -b -n -l -i project_id.bq_dataset.bg_table_prefix_
Migration Center C2C Data Import, v0.2
Customer: Demo Customer, Inc
Migration Center Reports directory: /home/user/mc-test/
BQ Table Prefix: bq_table_
Importing data into Big Query...
GCP Project ID: project_id
BQ Dataset Name: bq_dataset
Migration Center Data import...
Importing pricing report files...
Dataset project_id.bq_dataset already exists.
Importing mapped.csv into BQ Table: project_id.bq_dataset.bg_table_prefix_mapped
Loaded 14121165 rows and 27 columns to project_id.bq_dataset.bg_table_prefix_mapped
Importing unmapped.csv into BQ Table: project_id.bq_dataset.bg_table_prefix_unmapped
Loaded 63849667 rows and 17 columns to project_id.bq_dataset.bg_table_prefix_unmapped
Skipping discount.csv since there is no Migration Center data in the file.
Completed loading of Migration Center Data into Big Query.
Looker URL: https://lookerstudio.google.com/reporting/create?c.reportId=421c8150-e7ad-4190-b044-6a18ecdbd391&r.reportName=AWS+-%3E+GCP+Pricing+Analysis%3A+Demo+Customer%2C+Inc%2C+2025-01-21+18%3A19&ds.ds0.connector=bigQuery&ds.ds0.datasourceName=mapped&ds.ds0.projectId=project_id&ds.ds0.type=TABLE&ds.ds0.datasetId=bq_dataset&ds.ds0.tableId=bq_table_mapped&ds.ds1.connector=bigQuery&ds.ds1.datasourceName=unmapped&ds.ds1.projectId=project_id&ds.ds1.type=TABLE&ds.ds1.datasetId=bq_dataset&ds.ds1.tableId=bq_table_unmapped

Creating new Google Sheets...
Migration Center Sheets: https://docs.google.com/spreadsheets/d/123456789
```
