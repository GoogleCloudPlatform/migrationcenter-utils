# MC2BQ - Migration Center to BigQuery Exporter

This is a tool to allow Migration Center users to export the data and insights
from Migration Center to BigQuery to allow for advanced data analysis in BigQuery.

## Build from source

Simply use go build.

```sh
go build .
```

## Run locally

After you've built the CLI you can use it to export the data from your own computer.
Make sure you are [authenticated with gcloud](https://cloud.google.com/sdk/gcloud/reference/auth) and have permissions to view Migration Center data and edit BigQuery data.

```text
Usage: mc2bq [FLAGS...] <PROJECT> <DATASET> [TABLE-PREFIX]
Export Migration Center data to BigQuery

    PROJECT         Project you want to export Migration Center data from. (env: MC2BQ_PROJECT)
    DATASET         Dataset that will be used to store the tables in BigQuery. If a data set with that name does not exist, one will be created. (env: MC2BQ_DATASET)
    TABLE-PREFIX    A prefix to add to the table names, this can be done to store multiple exported tables in the same data set. (env: MC2BQ_TABLE_PREFIX)

  -dump-embedded-schema
        write the schema file embedded in the current version to stdout.
  -force
        force the export of the data even if the destination table exists, the operation will delete all the content in the original table. (env: MC2BQ_FORCE)
  -region string
        migration center region. (env: MC2BQ_REGION) (default "us-central1")
  -schema-path string
        use the schema at the specified path instead of using the embedded schema. (env: MC2BQ_SCHEMA_PATH)
  -target-project string
        target project where the data should be exported to, if not set the project that contains the migration center data will be used. (env: MC2BQ_TARGET_PROJECT)
  -version
        print the version and exit.
```

## Run in the cloud using Cloud Run

If you want to sync data periodically, you can set up a recurring Cloud Run job to do that.

### 1. Build an image

Because the schema might change in the future, it's recommended that you keep a copy of the schema and bake it in to your own image. That way you can upgrade the version of mc2bq in the image without updating the schema.

To do that, first copy the schema to the source directory (tools/mc2bq):

```sh
cd tools/mc2bq
cp pkg/schema/migrationcenter_v1_latest.schema.json migrationcenter_v1.schema.json
```

Build the image:

```sh
docker build . -t gcr.io/my-project/mc2bq:latest
```

And push it to the image repository:

```sh
docker push gcr.io/my-project/mc2bq:latest
```

### 2. Set-up a cloud run job

The simpleset way to set up a recurring export. See details on how to set that up [here](./terraform).

If you don't want to use terraform you can set-up the job manually.

1. **Create a service account** - The service account needs to have access to the following roles:
    * BigQuery Data Editor - To allow the creation of datasets and tables
    * BigQuery Job User - To allow accessing big query from a job and using the streaming API
    * Migration Center Viewer - To be able to read data from Migration Center

2. **Create a CloudRun Job** - The job should point to the image [created earlier](#1-build-an-image).

    Use either command line flags or environment variables to configure the execution.
    There is no need to set up the schema path as that is baked to the image.

3. **Set up cloud scheduler** - Use [cloud scheduler](https://cloud.google.com/scheduler) to set up a sync schedule.
