# Workday Data Export

This solution defines an Apache Airflow DAG (Directed Acyclic Graph) named "integration_template" that automates the deployment and execution of integrations for extracting data from Workday.

## DAG Configuration:

The Airflow DAG orchestrates the automated deployment and execution of integrations for extracting data from Workday. It uses Google Cloud services, Bash scripts, and a command-line tool (`integrationcli`) to manage and execute the integrations for different Workday entities.

- `YESTERDAY`: Defines the start date for DAG runs as yesterday's date.
- `default_args`: Sets default arguments for all tasks in the DAG, including owner, email notifications, retries, and start date.
- `models.DAG`: Defines the DAG itself with parameters like:
- `catchup`: Set to False to prevent backfilling historical data.
- `schedule_interval`: Determines how often the DAG runs (daily in this case).
- `params`: Provides a way to pass parameters to the DAG, including:
- `region`: The Google Cloud region for deployment.
- `project_id`: The Google Cloud project ID.
- `integration_name`: The base name for the integration.
- `ents`: A list of Workday entities to extract data from (e.g., "Disabilities", "Workers").

### DAG Tasks:

- **`get_access_token` (BashOperator):**
- Retrieves a Google Cloud access token using `gcloud auth print-access-token`.
- Pushes the access token to Airflow's XCom system for use by other tasks.

- **`deploy_workday_export_parent` (Python Task):**
- Iterates through each Workday entity specified in `ents`.
- For each entity, it deploys an integration named `workday-export-parent-{entity}`.
- This likely involves:
- Reading integration configuration files from Google Cloud Storage.
- Using `sed` to replace placeholders in the configuration with the current entity.
- Using `integrationcli` (a command-line tool) to apply the integration to the specified Google Cloud project.
- Each entity deployment is executed as a separate BashOperator task named `deploy_workday_export_parent_{entity}`.

- **`deploy_workday_export_page` (Python Task):**
- Similar to `deploy_workday_export_parent`, but it deploys integrations named `workday-export-page-{entity}`.
- Each entity deployment is executed as a separate BashOperator task named `deploy_workday_entity_export_page_{entity}`.

- **`exec_workday_parent` (Python Task):**
- Iterates through each Workday entity.
- For each entity, it executes an integration named `workday-export-parent-v1-{entity}`.
- The execution process involves similar steps as deployment: reading configuration, replacing placeholders, and using `integrationcli` to trigger the integration execution.
- Error handling is included to print error messages from the integration execution.
- Each entity execution is handled by a BashOperator task named `exec_workday_export_parent_{entity}`.

### DAG Task Dependencies:

- `get_access_token >> deploy_workday_export_parent() >> deploy_workday_export_page() >> exec_workday_parent()`: Defines the execution order of tasks. The access token is retrieved first, then parent integrations are deployed, followed by page integrations, and finally, parent integrations are executed.

### Dependencies:

- `gcloud`: Used to authenticate and get an access token.
- `integrationcli`: A command-line tool for managing integrations.
- Google Cloud Storage: Used to store integration configuration files.

## Application Integration JSON Definitions:

- The provided Application Integration JSON code snippets appear to define configuration for an event bus or workflow orchestration tool (it's not directly part of the Airflow DAG but is related). It includes:
- Trigger configurations for starting workflows based on API calls.
- Task configurations that define actions like running sub-workflows, connecting to data sources, field mapping, and data transfer to Google Cloud Storage.
- Integration parameters that define input and output data types and values used in the workflow.
