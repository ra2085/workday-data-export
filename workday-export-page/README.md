This integration fetches data from Workday, filters it based on selected columns, converts it to CSV format, and uploads it to Google Cloud Storage (GCS).

Here's a breakdown:

1. **Data Retrieval:** It connects to Workday and retrieves entities (data records) based on configurable parameters like page size, filter conditions, and sorting options.
2. **Data Transformation:** The retrieved data is processed to extract specific columns defined in the "select_columns" parameter. This filtered data is then transformed into CSV format.
3. **Data Storage:** The resulting CSV data is prepared for upload by setting the bucket name, folder path, and object name. Finally, it's uploaded to GCS.

**1. Trigger:**

* **Type**: API
* **Name**: workday-export-page
* **Action**: Triggers the integration when an API call is made to the defined endpoint.

**Tasks:**

**Prepare Entity Request (Task ID: 2):**

* **Type**: FieldMappingTask
* **Action**: Prepares the input for the Workday request by setting the `Task_1_listEntitiesPageToken` parameter. If it's the first page, the token is set to empty.
* **Input**:
* `Task_1_listEntitiesPageToken`: Token for pagination (initially empty)
* `empty_string`: Empty string literal
* **Output**:
* `Task_1_listEntitiesPageToken`: Updated pagination token

**Workday Request (Task ID: 1):**

* **Type**: GenericConnectorTask
* **Action**: Connects to a Workday instance and retrieves data.
* **Configuration**:
* `connectionName`: Name of the pre-configured Workday connection.
* `connectionVersion`: Version of the Workday connector being used.
* `operation`: Specifies the operation to perform (LIST_ENTITIES in this case).
* `entityType`: Type of Workday entity to retrieve (needs to be replaced with an actual entity).
* `listEntitiesPageSize`: Number of records to fetch per page.
* `listEntitiesSortByColumns`: Columns to sort the result by (optional).
* `filterClause`: Filter criteria for the data retrieval (optional).
* **Input**:
* `Task_1_listEntitiesPageToken`: Pagination token
* `Task_1_listEntitiesPageSize`: Page size
* `Task_1_listEntitiesSortByColumns`: Sort columns (optional)
* `Task_1_filterClause`: Filter criteria (optional)
* **Output**:
* `Task_1_connectorOutputPayload`: JSON payload containing the retrieved Workday data.
* `Task_1_listEntitiesNextPageToken`: Token for the next page of data (if any).

**Filter Columns (Task ID: 6):**

* **Type**: JsonnetMapperTask
* **Action**: Filters the Workday data, keeping only specified columns. Uses Jsonnet for data manipulation.
* **Input**:
* `Task_1_connectorOutputPayload`: Raw Workday data from the previous step.
* `select_columns`: Array of column names to keep in the output.
* **Output**:
* `WorkdayEntities`: Filtered Workday data in CSV format.

**Map to Output (Task ID: 4):**

* **Type**: FieldMappingTask
* **Action**: Maps the filtered Workday data and other parameters to the format required for GCS upload.
* **Input**:
* `WorkdayEntities`: Filtered Workday data in CSV format.
* `folder_name`: Folder within the GCS bucket where the file will be uploaded.
* **Output**:
* `Task_5_connectorInputPayload`: JSON object containing:
* `Bucket`: Name of the target GCS bucket.
* `FolderPath`: Path within the bucket to store the file.
* `ObjectName`: Generated filename (using UUID) with `.csv` extension.
* `Content`: Filtered Workday data in CSV format.

**To GCS (Task ID: 5):**

* **Type**: GenericConnectorTask
* **Action**: Uploads the prepared data to the specified GCS location.
* **Configuration**:
* `connectionName`: Name of the pre-configured GCS connection.
* `connectionVersion`: Version of the GCS connector.
* `operation`: Specifies the action to perform (EXECUTE_ACTION in this case).
* `actionName`: Name of the GCS action to execute (UploadObject).
* **Input**:
* `Task_5_connectorInputPayload`: Data to be uploaded, prepared in the previous step.
* **Output**:
* `Task_5_connectorOutputPayload`: Response from the GCS upload operation.

**Integration Parameters:**

* Defines various parameters used throughout the integration, including default values, data types, and producer/consumer relationships.

**Variable Masking:**

* Enables masking sensitive data in logs for security purposes.

**Overall Flow:**

1. The integration is triggered by an API call.
2. It fetches data from Workday, potentially in multiple pages.
3. The retrieved data is filtered to keep only relevant columns.
4. The filtered data is formatted and mapped for GCS upload.
5. The data is uploaded to the specified GCS location.