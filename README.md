# Workday Data Export

This solution uses Google's [Cloud Composer](https://cloud.google.com/composer) and [Application Integration](https://cloud.google.com/application-integration/docs/overview)
products to define an ETL pipeline for exporting data from a Workday tenant into Google Cloud Storage.

Cloud Composer is fully managed workflow orchestration service built on Apache Airflow.

The solution defines an Apache Airflow DAG (Directed Acyclic Graph) that automates the deployment and execution of Application Integrations for extracting data from Workday.

The raw logic for creating, scheduling, and monitoring the Application Integration jobs is written in Go.
The Go logic is built into a single binary. This binary can be run locally for testing or troubleshooting, or automated to run within a Kubernetes POD using Airflow.


## Airflow DAG Configuration

The Airflow DAG orchestrates the automated deployment and execution of integrations for extracting data from Workday.

- `YESTERDAY`: Defines the start date for DAG runs as yesterday's date.
- `default_args`: Sets default arguments for all tasks in the DAG, including owner, email notifications, retries, and start date.
- `models.DAG`: Defines the DAG itself with parameters like:
- `ENTITIES`:  List of Workday entities to export
- `PROJECT_ID`: The ID of the Google Cloud project where Integration and Connectors are running
- `REGION`: The Google Cloud region where the Integration is running on
- `WORKDAY_CONNECTION_NAME`: The connection name for the Workday Connector
- `WORKDAY_CONNECTION_REGION`: The region where the Workday connection is running on
- `GCS_BUCKET_NAME`: The name of the Google Cloud Storage bucket where the data is exported to
- `GCS_CONNECTION_NAME`: The connection name for the Google Cloud Storage Connector
- `GCS_CONNECTION_REGION`: The region where the Google Cloud Storage connection is running on

## Airflow DAG Tasks Design

![dag_graph](/docs/images/dag_graph.png)

- **`get_access_token` (BashOperator)**:
This task uses the `gcloud` CLI tool to get an OAuth2 access token and makes it available to subsequent tasks though the Airflow XCom system.


- **`export_entities` (Python Task)**:
This task iterates through the list of `ENTITIES`, and creates a Kubernetes POD definition for each entity.


- **`workday_export` (KubernetesPodOperator)**:
This is a dynamic / mapped task that creates a Kubernetes POD for each of the entities being exported.
Under the hood, there is a concurrent independent sub-task started for each of the entities being exported.
![mapped_tasks](/docs/images/mapped_tasks.png)
The POD itself is running the Go binary for creating, scheduling and monitoring the Application Integration jobs.

Error handling is done within the actual Kubernetes POD itself in the Go logic. When an error happens, the
export program panics, and the process exits with non-zero status code. This in turn causes the task itself to
be marked as failed.

Retries can be done by clearing  any individual task status. This will cause the Airflow scheduler to re-run the
task. Note, that you will also need to clear the status of the upstream `get_access_token` task to get a new
access token.


## Integration Design

At a high level, there are two integrations created for each entity that is being exported. There is a `parent` and
a `page` entity. The `parent` integration is responsible for looping through each page of data that needs to be exported for
a certain entity. The `page` integration is responsible for actually retrieving the data for each page, and storing it in GCS.


## Application Integration JSON Definitions

The provided Application Integration JSON code snippets to define configuration for an event bus or workflow orchestration tool (it's not directly part of the Airflow DAG but is related). 

It includes:
- Trigger configurations for starting workflows based on API calls.
- Task configurations that define actions like running sub-workflows, connecting to data sources, field mapping, and data transfer to Google Cloud Storage.
- Integration parameters that define input and output data types and values used in the workflow.
