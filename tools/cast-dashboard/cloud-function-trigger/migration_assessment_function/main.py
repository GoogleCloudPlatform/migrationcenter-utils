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

import functions_framework
import os
import json
import requests
import google.oauth2.id_token
import google.auth.transport.requests
from google.cloud import migrationcenter_v1
import csv
from io import StringIO
import pandas as pd
import gcsfs
import google.cloud.logging
import logging

from google.cloud import storage

# Triggered by a change in a storage bucket
@functions_framework.cloud_event
def create_mc_group(cloud_event):

    client = google.cloud.logging.Client()
    client.setup_logging()

    data = cloud_event.data

    bucket = data["bucket"]
    file_name = data["name"]

    if file_name != "analysisResults.txt":
        logging.error("ERROR : File name should be surveyResults and format should either be CSV or comma separated txt only")
        exit(1)
        
    print(f"filename = {file_name}")
    project_name = os.environ.get('PROJECT_NAME')
    mig_parent = os.environ.get('MIGRATIONCENTER_PATH')
    fs = gcsfs.GCSFileSystem(project=project_name)
    fs.invalidate_cache()
    with fs.open(f"{bucket}/{file_name}") as f:
        df = pd.read_csv(f)

        groups = list(df["Application"].unique())

        print(f"GROUPS: {groups}")
    
    client = migrationcenter_v1.MigrationCenterClient()

    for groupname in groups:

        mc_group_id = (groupname).lower().replace(" ","-").replace("_","-")
        groupdetails = migrationcenter_v1.Group({"display_name":mc_group_id})

        print(f"GROUP ID: {mc_group_id}")
        if len(mc_group_id) < 4 or  len(mc_group_id) > 63:
            continue
        request = migrationcenter_v1.CreateGroupRequest(
            parent=mig_parent,
            group_id=mc_group_id,
            group=groupdetails
        )

        operation = client.create_group(request=request)
        response = operation.result()
        print(response)
    