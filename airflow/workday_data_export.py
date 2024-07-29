# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import datetime
import os

from airflow import models
from airflow.operators import bash
from airflow.operators.python import PythonOperator
from airflow.providers.cncf.kubernetes.operators.pod import KubernetesPodOperator

YESTERDAY = datetime.datetime.now() - datetime.timedelta(days=1)
project_id = os.getenv("GCP_PROJECT")

default_args = {
    "owner": "Miguel Mendoza (Google)",
    "depends_on_past": False,
    "email": ["miguelmendoza@google.com"],
    "email_on_failure": False,
    "email_on_retry": False,
    "retries": 0,
    "retry_delay": datetime.timedelta(minutes=5),
    "start_date": YESTERDAY,
}

with models.DAG(
        "workday_data_export",
        catchup=False,
        default_args=default_args,
        schedule_interval=datetime.timedelta(days=1),
        params={
            "ENTITIES": ["Disabilities", "Workers", "JobProfiles", "Genders"],
            "PROJECT_ID": f"{project_id}",
            "REGION": "us-west2",
            "WORKDAY_CONNECTION_NAME": "workday",
            "WORKDAY_CONNECTION_REGION": "us-west2",
            "GCS_BUCKET_NAME": "miguel-embeddings-storage",
            "GCS_CONNECTION_NAME": "miguel-embeddings-storage",
            "GCS_CONNECTION_REGION": "us-west2"
        },
) as dag:
    get_access_token = bash.BashOperator(
        task_id="get_access_token", bash_command="gcloud auth print-access-token", do_xcom_push=True
    )

    def get_entities(**kwargs):
        entities = kwargs['params']['ENTITIES']
        dag.log.info(entities)
        pods = []
        for entity in entities:
            dag.log.info(f"Setting up pod for {entity} ...")
            pods.append({
                "image": f'gcr.io/{project_id}/workday-export',
                "name": entity.lower(),
                "get_logs": True,
                "env_vars": {
                    "TOKEN": "{{ ti.xcom_pull(task_ids='get_access_token') }}",
                    "PROJECT_ID": "{{ params.PROJECT_ID }}",
                    "REGION": "{{ params.REGION }}",
                    "ENTITY": entity,
                    "WORKDAY_CONNECTION_NAME": "{{ params.WORKDAY_CONNECTION_NAME }}",
                    "WORKDAY_CONNECTION_REGION": "{{ params.WORKDAY_CONNECTION_REGION }}",
                    "GCS_BUCKET_NAME": "{{ params.GCS_BUCKET_NAME }}",
                    "GCS_CONNECTION_NAME": "{{ params.GCS_CONNECTION_NAME }}",
                    "GCS_CONNECTION_REGION": "{{ params.GCS_CONNECTION_REGION }}",
                }
            })
        return pods

    run_get_entities = PythonOperator(task_id="get_entities", python_callable=get_entities)
    run_export_entities = KubernetesPodOperator.partial(task_id="export_entities").expand_kwargs(run_get_entities.output)

    get_access_token >> run_export_entities
