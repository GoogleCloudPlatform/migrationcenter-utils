# calcctl to Sheets

This application will automatically create a Google Sheets from a [calcctl](http://go/calcctl) generated report.


---
### Google Cloud SDK

In order for calcctl to Sheets to authenticate against Google, you must have the Google Cloud SDK (gcloud) installed.

Instructions for installing the Google Cloud SDK can be found in the [Install the gcloud CLI](https://cloud.google.com/sdk/docs/install) guide.

---
### Python Environment

calcctl to Sheets requires Python3 to be installed. Once it is installed, you can install the required Python modules using the included `requirements.txt` file:

```shell
$ cd calcctl-to-sheets/python
$ pip3 install -r requirements.txt
```
#### Using Cloud Top with virtual environment 

Google Cloud Top instances have restrictions with installing python modules, so you must run the python script inside of a virtual environment:

```shell
$ sudo apt install python3.11-venv
$ cd calcctl-to-sheets/python
$ python3 -m venv ../venv
$ source ../venv/bin/activate
(venv) $ pip3 install -r requirements.txt
```

Now you can run the python script anytime by switching to the virtual environment:

```shell
$ cd calcctl-to-sheets/python
$ source ../venv/bin/activate
(venv) $ python3 calcctl-to-sheets.py ....
```

---
### Authenticate to Google

In order to use the Google Drive/Sheets API, you must have a Google project setup.
Once you have a project setup, you can run the following to authenticate against the Google project using your Google account & run calcctl to Sheets:

```shell
$ gcloud auth login
$ gcloud config set project <PROJECT-ID>
$ gcloud services enable drive.googleapis.com sheets.googleapis.com
$ gcloud auth application-default login --scopes='https://www.googleapis.com/auth/drive','https://www.googleapis.com/auth/cloud-platform'
```

---
#### Example calcctl to Sheets Run


```shell 
$ cd calcctl-to-sheets/python
$ python3 calcctl-to-sheets.py -d ~/calcctl/reports/ -c "Demo Customer, Inc"
calcctl to Google sheets, v0.1
Customer: Demo Customer, Inc
Calcctl reports directory: /Users/demo/calcctl/reports/
Creating new Google Sheets...
Importing calcctl files...
calcctl Report: Demo Customer, Inc: https://docs.google.com/spreadsheets/d/1234567890
```
