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

from airflow import models
from airflow.operators import bash
from airflow.models import Variable
from airflow.decorators import task

YESTERDAY = datetime.datetime.now() - datetime.timedelta(days=1)

default_args = {
    "owner": "Ruben Gonzalez (Google)",
    "depends_on_past": False,
    "email": ["gonzalezruben@google.com"],
    "email_on_failure": False,
    "email_on_retry": False,
    "retries": 0,
    "retry_delay": datetime.timedelta(minutes=5),
    "start_date": YESTERDAY,
}

with models.DAG(
    "integration_template",
    catchup=False,
    default_args=default_args,
    schedule_interval=datetime.timedelta(days=1),
    params={"region":"us-west1","project_id":"cymbal-ai", "integration_name":"workday-export-parent", "ents": ["Disabilities", "Workers", "JobProfiles", "Genders"]},
) as dag:

    get_access_token = bash.BashOperator(
        task_id="get_access_token", bash_command="gcloud auth print-access-token", do_xcom_push=True
    )

    @task
    def deploy_workday_export_parent(**context):
        from airflow.models import Variable
        entities=context['params']['ents']
        project_id=context['params']['project_id']
        region=context['params']['region']
        integration_cli=Variable.get("integration_cli")
        access_token=context['ti'].xcom_pull(task_ids='get_access_token')
        for entity in entities:
            bash_cmd=f'''
            cd $HOME &&
            { integration_cli }
            rm -rf ./workday-export-parent-$ENTITY/ && cp /home/airflow/gcs/data/workday-export-parent/ ./workday-export-parent-$ENTITY -r && ls && cd ./workday-export-parent-$ENTITY/src &&
            sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./workday-export-parent-v1.json && mv ./workday-export-parent-v1.json ./workday-export-parent-v1-$ENTITY.json && cd .. && cd ./dev/overrides &&
            sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./overrides.json && cd ../../../ &&
            integrationcli integrations apply -e dev -f ./workday-export-parent-$ENTITY/ -p { project_id } -r { region } -t { access_token } &&
            rm -r ./workday-export-parent-$ENTITY/
            '''
            task_ident = f"deploy_workday_export_parent_{entity}"
            deploy_entity_page = bash.BashOperator(
                task_id=task_ident,
                bash_command=bash_cmd,
                env={"ENTITY":entity,"HOME":"/home/airflow"}
            )
            deploy_entity_page.execute(dict())

    @task
    def deploy_workday_export_page(**context):
        from airflow.models import Variable
        entities=context['params']['ents']
        project_id=context['params']['project_id']
        region=context['params']['region']
        integration_cli=Variable.get("integration_cli")
        access_token=context['ti'].xcom_pull(task_ids='get_access_token')
        for entity in entities:
            bash_cmd=f'''
            echo $HOME && cd $HOME &&
            { integration_cli }
            rm -rf ./workday-export-page-$ENTITY/ && cp /home/airflow/gcs/data/workday-export-page/ ./workday-export-page-$ENTITY -r && ls && cd ./workday-export-page-$ENTITY/src &&
            sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./workday-export-page.json && mv ./workday-export-page.json ./workday-export-page-$ENTITY.json && cd .. && cd ./dev/overrides &&
            sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./overrides.json && cd ../../../ &&
            integrationcli integrations apply -e dev -f ./workday-export-page-$ENTITY/ -p { project_id } -r { region } -t { access_token } &&
            rm -r ./workday-export-page-$ENTITY/
            '''
            task_ident = f"deploy_workday_entity_export_page_{entity}"
            deploy_entity_page = bash.BashOperator(
                task_id=task_ident,
                bash_command=bash_cmd,
                env={"ENTITY":entity,"HOME":"/home/airflow"}
            )
            deploy_entity_page.execute(dict())

    @task
    def exec_workday_parent(**context):
        from airflow.models import Variable
        entities=context['params']['ents']
        project_id=context['params']['project_id']
        region=context['params']['region']
        integration_cli=Variable.get("integration_cli")
        access_token=context['ti'].xcom_pull(task_ids='get_access_token')
        for entity in entities:
            bash_cmd=f'''
            cd $HOME &&
            { integration_cli }
            rm -rf ./workday-export-parent-$ENTITY/ && cp /home/airflow/gcs/data/workday-export-parent/ ./workday-export-parent-$ENTITY -r && ls && cd ./workday-export-parent-$ENTITY/src &&
            sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./workday-export-parent-v1.json && mv ./workday-export-parent-v1.json ./workday-export-parent-v1-$ENTITY.json && cd .. && cd ./dev/overrides &&
            sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./overrides.json && sed -i "s/REPLACEWITHENTITY/$ENTITY/g" ./exec.json  && cd ../../../ &&
            integrationcli integrations execute -n workday-export-parent-v1-$ENTITY -f ./workday-export-parent-$ENTITY/dev/overrides/exec.json -p { project_id } -r { region } -t { access_token } &&
            rm -r ./workday-export-parent-$ENTITY/
            '''
            task_ident = f"exec_workday_export_parent_{entity}"
            deploy_entity_page = bash.BashOperator(
                task_id=task_ident,
                bash_command=bash_cmd,
                env={"ENTITY":entity,"HOME":"/home/airflow"}
            )
            try:
                deploy_entity_page.execute(dict())
            except Exception as e:
                for e in job.errors:
                    # apply your logic
                    print('ERROR: {}'.format(e['message']))

    get_access_token >> deploy_workday_export_parent() >> deploy_workday_export_page() >> exec_workday_parent()
