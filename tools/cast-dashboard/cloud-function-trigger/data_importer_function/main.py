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

from google.cloud import bigquery
import os
import base64
import json
import logging

def import_csv_to_bigquery(event, context):
    logging.info(f"Received event: {event}")

    # Decode the data from the event
    data = base64.b64decode(event['data']).decode('utf-8')
    logging.info(f"Decoded data: {data}")

    # Load the data into a JSON object
    data_json = json.loads(data)
    logging.info(f"JSON data: {data_json}")

    name = data_json.get('name')
    if not name:
        logging.error("No name found in the event data")
        return

    logging.info(f"Processing file: {name}")

    client = bigquery.Client()
    bucket_name = os.environ.get('BUCKET_NAME')
    dataset_id = os.environ.get('BIGQUERY_DATASET')
    table_id = os.environ.get('BIGQUERY_TABLE')

    table_ref = client.dataset(dataset_id).table(table_id)
    job_config = bigquery.LoadJobConfig(
        source_format=bigquery.SourceFormat.CSV,
        skip_leading_rows=1,
        autodetect=True,
    )

    uri = f"gs://{bucket_name}/{name}"
    logging.info(f"URI: {uri}")

    load_job = client.load_table_from_uri(uri, table_ref, job_config=job_config)
    load_job.result()  # Waits for the job to complete

    logging.info(f"Loaded {load_job.output_rows} rows into {dataset_id}:{table_id} from {uri}.")
