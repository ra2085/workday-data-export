package util

import (
	"bufio"
	"context"
	"fmt"
	"github.com/go-errors/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/connectors/v1"
	"google.golang.org/api/integrations/v1"
	"google.golang.org/api/option"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func WriteOutputText(output string, outputText []byte) error {
	var err error

	if output == "-" || len(output) == 0 {
		fmt.Printf("%s", string(outputText))
		return nil
	}

	dir := filepath.Dir(output)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return errors.New(err)
	}

	err = os.WriteFile(output, outputText, os.ModePerm)
	if err != nil {
		return errors.New(err)
	}
	return nil
}

func PushDir(dir string) func() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	err = os.Chdir(dir)
	if err != nil {
		panic(err)
	}

	popDir := func() {
		Must(os.Chdir(wd))
	}

	return popDir
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Run(cmd *exec.Cmd, logger *log.Logger) {
	r, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	done := make(chan struct{})
	scanner := bufio.NewScanner(r)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			logger.Println(line)
		}
		done <- struct{}{}
	}()

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	<-done
	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
}

func RequireArg(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

func GetIntegrationsSvc(token string) (*integrations.Service, error) {
	var err error
	bCtx := context.Background()

	config := &oauth2.Config{}
	t := new(oauth2.Token)
	t.AccessToken = token

	var svc *integrations.Service
	if svc, err = integrations.NewService(bCtx, option.WithTokenSource(config.TokenSource(bCtx, t))); err != nil {
		return nil, errors.New(err)
	}
	return svc, nil
}

func GetConnectorsSvc(token string) (*connectors.Service, error) {
	var err error
	bCtx := context.Background()

	config := &oauth2.Config{}
	t := new(oauth2.Token)
	t.AccessToken = token

	var svc *connectors.Service
	if svc, err = connectors.NewService(bCtx, option.WithTokenSource(config.TokenSource(bCtx, t))); err != nil {
		return nil, errors.New(err)
	}
	return svc, nil
}
