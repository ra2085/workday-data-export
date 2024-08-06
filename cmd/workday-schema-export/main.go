package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"google.golang.org/api/connectors/v1"
	"log"
	"os"
	"sync"
	"time"
	"workday-data-export/pkg/util"
)

var ProjectId string = os.Getenv("PROJECT_ID")
var Token = os.Getenv("TOKEN")
var WorkdayConnectionName = os.Getenv("WORKDAY_CONNECTION_NAME")
var WorkdayConnectionRegion = os.Getenv("WORKDAY_CONNECTION_REGION")

type Context struct {
	Entity                  string
	ProjectId               string
	WorkdayConnectionName   string
	WorkdayConnectionRegion string
}

var requestCount int

func main() {
	var err error
	logger := log.New(os.Stdout, fmt.Sprintf("%s - ", "workday-export"), log.Flags())

	util.RequireArg(ProjectId != "", "PROJECT_ID env var is required")
	util.RequireArg(WorkdayConnectionRegion != "", "WORKDAY_CONNECTION_REGION env var is required")
	util.RequireArg(WorkdayConnectionName != "", "WORKDAY_CONNECTION_NAME env var is required")
	util.RequireArg(Token != "", "TOKEN env var is required")

	var svc *connectors.Service
	if svc, err = util.GetConnectorsSvc(Token); err != nil {
		panic(err)
	}

	entities, err := getEntityTypesList(svc, Context{
		ProjectId:               ProjectId,
		WorkdayConnectionName:   WorkdayConnectionName,
		WorkdayConnectionRegion: WorkdayConnectionRegion,
	}, logger)

	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	maxConcurrent := 25
	ch := make(chan int, maxConcurrent)

	for _, entity := range entities {
		if entity == "" {
			continue
		}

		requestCount += 2
		if float32(requestCount) >= 1000*0.9 {
			fmt.Printf("**** Sleeping for 1 minute to stay under the GlobalRequestCount rate limit *** \n")
			time.Sleep(1 * time.Minute) //not ideal, but will do
			requestCount = 0
		}

		wg.Add(1)
		ch <- 1

		go func() {
			defer func() { wg.Done(); <-ch }()

			ctx := Context{
				Entity:                  entity,
				ProjectId:               ProjectId,
				WorkdayConnectionName:   WorkdayConnectionName,
				WorkdayConnectionRegion: WorkdayConnectionRegion,
			}
			if err = exportEntitySchema(svc, ctx, logger); err != nil {
				logger.Printf("*** Error retrieving %s entity schema ...", entity)
				logger.Printf("*** ERROR: %s", err.Error())
			}

		}()
	}
	wg.Wait()
}

func getEntityTypesList(svc *connectors.Service, ctx Context, logger *log.Logger) ([]string, error) {
	logger.Printf("**** Requesting list of entities  ***")

	var err error
	var entityTypes []string

	connection := fmt.Sprintf("projects/%s/locations/%s/connections/%s", ctx.ProjectId, ctx.WorkdayConnectionRegion, ctx.WorkdayConnectionName)
	listCall := svc.Projects.Locations.Connections.ConnectionSchemaMetadata.ListEntityTypes(fmt.Sprintf("%s/connectionSchemaMetadata", connection))
	nextPageToken := ""
	var entityTypesRes *connectors.ListEntityTypesResponse
	for {
		if entityTypesRes, err = listCall.PageSize(500).PageToken(nextPageToken).Do(); err != nil {
			return nil, errors.New(err)
		}

		for _, entityType := range entityTypesRes.EntityTypes {
			entityTypes = append(entityTypes, entityType.Entity)
		}

		nextPageToken = entityTypesRes.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	logger.Printf("**** Found %d entity types in conection metadata  ***", len(entityTypes))

	return entityTypes, nil
}

func exportEntitySchema(svc *connectors.Service, ctx Context, logger *log.Logger) error {
	var err error

	connection := fmt.Sprintf("projects/%s/locations/%s/connections/%s", ctx.ProjectId, ctx.WorkdayConnectionRegion, ctx.WorkdayConnectionName)

	getEntityOperation := svc.Projects.Locations.Connections.ConnectionSchemaMetadata.GetEntityType(fmt.Sprintf("%s/connectionSchemaMetadata", connection))

	var getEntityOperationRes *connectors.Operation
	if getEntityOperationRes, err = getEntityOperation.EntityId(ctx.Entity).Do(); err != nil {
		return errors.New(err)
	}

	attempts := 0
	maxAttempts := 3
	retryDelay := 5 * time.Second

	for {

		time.Sleep(retryDelay)
		retryDelay = retryDelay * 2

		attempts += 1
		logger.Printf("*** Requesting %s entity schema (attempt %d / %d) ...", ctx.Entity, attempts, maxAttempts)
		if getEntityOperationRes, err = svc.Projects.Locations.Operations.Get(getEntityOperationRes.Name).Do(); err != nil {
			return errors.New(err)
		}

		if getEntityOperationRes.Done {
			break
		}

		if attempts == maxAttempts {
			break
		}

	}

	if !getEntityOperationRes.Done {
		return errors.New("could not retrieve entity from connection metadata")
	}

	logger.Printf("*** Saving %s entity schema ...", ctx.Entity)

	if err = os.MkdirAll("schemas", os.ModePerm); err != nil {
		return errors.New(err)
	}

	//save as normal JSON
	var jsonText []byte

	if jsonText, err = getEntityOperationRes.Response.MarshalJSON(); err != nil {
		return errors.New(err)
	}

	if len(jsonText) == 0 {
		return errors.Errorf("no schema found for %s entity", ctx.Entity)
	}

	jsonMap := map[string]any{}
	if err = json.Unmarshal(jsonText, &jsonMap); err != nil {
		return errors.New(err)
	}

	if jsonText, err = json.MarshalIndent(jsonMap, "", "  "); err != nil {
		return errors.New(err)
	}

	if err = util.WriteOutputText(fmt.Sprintf("schemas/%s.json", ctx.Entity), jsonText); err != nil {
		return err
	}

	//save as BigQuery JSON
	var bqFields []BigQueryField
	if bqFields, err = convertSchemaToBigQuery(jsonText); err != nil {
		return err
	}

	var bqText []byte
	if bqText, err = json.MarshalIndent(bqFields, "", "  "); err != nil {
		return errors.New(err)
	}

	if err = util.WriteOutputText(fmt.Sprintf("schemas/%s.bq.json", ctx.Entity), bqText); err != nil {
		return err
	}

	return nil
}

type BigQueryField struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Mode string `json:"mode"`
}

func convertSchemaToBigQuery(schemaJSON []byte) ([]BigQueryField, error) {
	schema := &connectors.RuntimeEntitySchema{}
	var err error
	if err = json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, errors.New(err)
	}

	var bqFields []BigQueryField
	for _, field := range schema.Fields {

		fieldMode := "REQUIRED"
		if field.Nullable {
			fieldMode = "NULLABLE"
		}

		fieldType := "STRING"

		switch field.DataType {
		case "DATA_TYPE_BOOLEAN":
			fieldType = "BOOLEAN"
		case "DATA_TYPE_DECIMAL":
			fieldType = "DECIMAL"
		case "DATA_TYPE_TIMESTAMP":
			fieldType = "TIMESTAMP"
		case "DATA_TYPE_DATE":
			fallthrough
		case "DATA_TYPE_TIME":
			fallthrough
		case "DATA_TYPE_VARCHAR":
			fallthrough
		default:
			fieldType = "STRING"
		}

		bqFields = append(bqFields, BigQueryField{
			Name: field.Field,
			Type: fieldType,
			Mode: fieldMode,
		})
	}

	return bqFields, nil

}
