package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"google.golang.org/api/connectors/v1"
	"google.golang.org/api/googleapi"
	"log"
	"os"
	"strings"
	"workday-data-export/pkg/util"
)

var ProjectId string = os.Getenv("PROJECT_ID")
var Token = os.Getenv("TOKEN")
var Entities = os.Getenv("ENTITIES")
var WorkdayConnectionName = os.Getenv("WORKDAY_CONNECTION_NAME")
var WorkdayConnectionRegion = os.Getenv("WORKDAY_CONNECTION_REGION")

type Context struct {
	Entity                  string
	ProjectId               string
	WorkdayConnectionName   string
	WorkdayConnectionRegion string
}

func main() {
	var err error
	logger := log.New(os.Stdout, fmt.Sprintf("%s - ", "workday-export"), log.Flags())

	util.RequireArg(ProjectId != "", "PROJECT_ID env var is required")
	util.RequireArg(WorkdayConnectionRegion != "", "WORKDAY_CONNECTION_REGION env var is required")
	util.RequireArg(WorkdayConnectionName != "", "WORKDAY_CONNECTION_NAME env var is required")
	util.RequireArg(Token != "", "TOKEN env var is required")
	util.RequireArg(Entities != "", "ENTITIES env var is required")

	entitiesSlice := strings.Split(Entities, ",")
	for _, entity := range entitiesSlice {
		if entity == "" {
			continue
		}
		ctx := Context{
			Entity:                  entity,
			ProjectId:               ProjectId,
			WorkdayConnectionName:   WorkdayConnectionName,
			WorkdayConnectionRegion: WorkdayConnectionRegion,
		}
		if err = exportEntitySchema(ctx, logger); err != nil {
			panic(err)
		}
	}
}

func exportEntitySchema(ctx Context, logger *log.Logger) error {
	location := fmt.Sprintf("schemas/%s.json", ctx.Entity)
	logger.Printf("**** Exporting Schema for %s entity to %s ***", ctx.Entity, location)

	var svc *connectors.Service
	var err error
	if svc, err = util.GetConnectorsSvc(Token); err != nil {
		return err
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/connections/%s", ctx.ProjectId, ctx.WorkdayConnectionRegion, ctx.WorkdayConnectionName)
	listCall := svc.Projects.Locations.Connections.RuntimeEntitySchemas.List(parent)

	var listResponse *connectors.ListRuntimeEntitySchemasResponse
	if listResponse, err = listCall.PageToken("").PageSize(1).Filter(fmt.Sprintf("entity=\"%s\"", ctx.Entity)).Do(); err != nil {
		if gErr, ok := err.(*googleapi.Error); ok {
			fmt.Println("Status code: %v", gErr.Code)
		}
		return errors.New(err)
	}

	jsonText, err := json.Marshal(listResponse)
	if err = os.MkdirAll("schemas", os.ModePerm); err != nil {
		return errors.New(err)
	}

	if err = util.WriteOutputText(location, jsonText); err != nil {
		return err
	}

	return nil
}
