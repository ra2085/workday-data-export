package main

import (
	"bytes"
	"fmt"
	"github.com/go-errors/errors"
	cp "github.com/otiai10/copy"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/integrations/v1"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
	"workday-data-export/pkg/util"
)

var showStack string = "true"

var ProjectId string = os.Getenv("PROJECT_ID")
var Region string = os.Getenv("REGION")
var Token = os.Getenv("TOKEN")
var Entity = os.Getenv("ENTITY")
var EntitiesPerPage = os.Getenv("ENTITIES_PER_PAGE")
var GcsConnectionName = os.Getenv("GCS_CONNECTION_NAME")
var GcsConnectionRegion = os.Getenv("GCS_CONNECTION_REGION")
var GcsBucketName = os.Getenv("GCS_BUCKET_NAME")
var WorkdayConnectionName = os.Getenv("WORKDAY_CONNECTION_NAME")
var WorkdayConnectionRegion = os.Getenv("WORKDAY_CONNECTION_REGION")

type Context struct {
	Entity                  string
	ProjectId               string
	Region                  string
	GCSConnectionName       string
	GCSConnectionRegion     string
	GCSBucketName           string
	WorkdayConnectionName   string
	WorkdayConnectionRegion string
	EntitiesPerPage         string
}

func main() {
	var err error
	logger := log.New(os.Stdout, fmt.Sprintf("%s - ", "workday-export"), log.Flags())

	util.RequireArg(ProjectId != "", "PROJECT_ID env var is required")
	util.RequireArg(Region != "", "REGION env var is required")
	util.RequireArg(Token != "", "TOKEN env var is required")
	util.RequireArg(Entity != "", "ENTITY env var is required")
	util.RequireArg(EntitiesPerPage != "", "ENTITIES_PER_PAGE env var is required")
	util.RequireArg(GcsConnectionName != "", "GCS_CONNECTION_NAME env var is required")
	util.RequireArg(GcsConnectionRegion != "", "GCS_CONNECTION_REGION env var is required")
	util.RequireArg(GcsBucketName != "", "GCS_BUCKET_NAME env var is required")
	util.RequireArg(WorkdayConnectionName != "", "WORKDAY_CONNECTION_NAME env var is required")
	util.RequireArg(WorkdayConnectionRegion != "", "WORKDAY_CONNECTION_REGION env var is required")

	ctx := Context{
		Entity:                  Entity,
		ProjectId:               ProjectId,
		Region:                  Region,
		GCSConnectionName:       GcsConnectionName,
		GCSConnectionRegion:     GcsConnectionRegion,
		GCSBucketName:           GcsBucketName,
		WorkdayConnectionName:   WorkdayConnectionName,
		WorkdayConnectionRegion: WorkdayConnectionRegion,
		EntitiesPerPage:         EntitiesPerPage,
	}

	if err = createIntegration("workday-export-parent", ctx, logger); err != nil {
		panic(err)
	}

	if err = createIntegration("workday-export-page", ctx, logger); err != nil {
		panic(err)
	}

	var executionId string
	if executionId, err = scheduleIntegration(ctx, logger); err != nil {
		panic(err)
	}

	if err = waitForIntegrationExecution(executionId, ctx, logger); err != nil {
		panic(err)
	}

}

func waitForIntegrationExecution(executionID string, ctx Context, logger *log.Logger) error {
	logger.Printf("**** Waiting for %s integration to complete ***", ctx.Entity)

	var svc *integrations.Service
	var err error
	if svc, err = util.GetIntegrationsSvc(Token); err != nil {
		return err
	}

	name := fmt.Sprintf("projects/%s/locations/%s/integrations/workday-export-parent-v1-%s/executions/%s", ctx.ProjectId, ctx.Region, ctx.Entity, executionID)

	var state string
	for {
		getExecution := svc.Projects.Locations.Integrations.Executions.Get(name)

		var executionResponse *integrations.GoogleCloudIntegrationsV1alphaExecution
		if executionResponse, err = getExecution.Do(); err != nil {
			if gErr, ok := err.(*googleapi.Error); ok {
				fmt.Println("Status code: %v", gErr.Code)
			}
			return errors.New(err)
		}

		state = executionResponse.ExecutionDetails.State
		logger.Printf("Execution state is: %s ... ", state)

		if state != "PENDING" && state != "PROCESSING" && state != "STATE_UNSPECIFIED" {
			break
		}
		time.Sleep(10 * time.Second)
	}

	if state != "SUCCEEDED" {
		return errors.Errorf("Integration exited with %s state", state)
	}

	return nil

}

func createIntegration(integrationDir string, ctx Context, logger *log.Logger) error {
	logger.Printf("**** Deploying Integration (%s) for %s entity ***", integrationDir, ctx.Entity)
	var tmpDir string
	var err error
	if tmpDir, err = os.MkdirTemp("", "integration"); err != nil {
		return errors.New(err)
	}
	logger.Printf("dir: %s\n", tmpDir)

	if err = cp.Copy(filepath.Join("integrations", integrationDir), filepath.Join(tmpDir, integrationDir)); err != nil {
		return errors.New(err)
	}

	//replace all template place-holders
	if err = replaceAll(tmpDir, ctx); err != nil {
		return err
	}

	//change the name of the integration
	parentDir := filepath.Join(tmpDir, integrationDir, "src")
	var entries []fs.DirEntry
	if entries, err = os.ReadDir(parentDir); err != nil {
		return errors.New(err)
	}

	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if ext == ".json" {
			oldName := entry.Name()
			newName := strings.ReplaceAll(entry.Name(), ".json", fmt.Sprintf("-%s.json", ctx.Entity))
			if err = os.Rename(filepath.Join(parentDir, oldName), filepath.Join(parentDir, newName)); err != nil {
				return errors.New(err)
			}
		}
	}

	//use the integrationcli binary to create the integration
	popd := util.PushDir(tmpDir)
	defer popd()
	integrationCli := exec.Command("integrationcli", "integrations", "apply",
		"-f", integrationDir,
		"-p", ctx.ProjectId,
		"-r", ctx.Region,
		"-t", Token,
	)
	util.Run(integrationCli, logger)

	return nil
}

func replaceAll(dir string, ctx Context) error {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		tmpl, err := template.New(filepath.Base(path)).ParseFiles(path)
		if err != nil {
			return errors.New(err)
		}

		renderedBytes := bytes.Buffer{}
		if err = tmpl.Execute(&renderedBytes, ctx); err != nil {
			return errors.New(err)
		}

		if err = util.WriteOutputText(path, renderedBytes.Bytes()); err != nil {
			return errors.New(err)
		}

		return nil
	})

	if err != nil {
		return errors.New(err)
	}
	return nil
}

func scheduleIntegration(ctx Context, logger *log.Logger) (string, error) {
	logger.Printf("**** Scheduling Integration for %s entity ***", ctx.Entity)

	var svc *integrations.Service
	var err error
	if svc, err = util.GetIntegrationsSvc(Token); err != nil {
		return "", err
	}

	var entitiesPerPage int
	if entitiesPerPage, err = strconv.Atoi(ctx.EntitiesPerPage); err != nil {
		return "", errors.New(err)
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/integrations/workday-export-parent-v1-%s", ctx.ProjectId, ctx.Region, ctx.Entity)
	scheduleCall := svc.Projects.Locations.Integrations.Schedule(parent, &integrations.GoogleCloudIntegrationsV1alphaScheduleIntegrationsRequest{
		InputParameters: map[string]integrations.GoogleCloudIntegrationsV1alphaValueType{
			"page_size": {
				IntValue: int64(entitiesPerPage),
			},
			"select_columns": {
				StringArray: &integrations.GoogleCloudIntegrationsV1alphaStringParameterArray{
					StringValues: []string{},
				},
			},
			"folder_name": {
				StringValue: ctx.Entity,
			},
		},
		TriggerId: fmt.Sprintf("api_trigger/get-workday-entities-%s", ctx.Entity),
	})

	var scheduleResponse *integrations.GoogleCloudIntegrationsV1alphaScheduleIntegrationsResponse
	if scheduleResponse, err = scheduleCall.Do(); err != nil {
		if gErr, ok := err.(*googleapi.Error); ok {
			fmt.Println("Status code: %v", gErr.Code)
		}
		return "", errors.New(err)
	}

	if len(scheduleResponse.ExecutionInfoIds) != 1 {
		return "", errors.Errorf("expected exactly one executionId in response, but got: %d", len(scheduleResponse.ExecutionInfoIds))
	}

	executionId := scheduleResponse.ExecutionInfoIds[0]

	logger.Printf("Integration scheduled successfully - executionId: %s", executionId)

	return executionId, nil
}
