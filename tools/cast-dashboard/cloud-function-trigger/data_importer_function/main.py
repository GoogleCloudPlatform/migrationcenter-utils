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
