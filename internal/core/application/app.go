/*
Copyright 2023 - PRESENT Meltred

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package application

import (
	"bytes"
	"context"
	"errors"
	"meltred/meltcd/spec"
	"time"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"gopkg.in/yaml.v2"
)

type Application struct {
	Name         string `json:"name"`
	Source       Source `json:"source"`
	RefreshTimer string `json:"refresh_timer"` // Timer to check for Sync format of "3m50s"
	Health       Health `json:"health"`
	LiveState    string `json:"live_state"`
}

type Health int

const (
	Healthy Health = iota
	Progressing
	Degraded
	Suspended
)

func New(spec Spec) Application {
	return Application{
		Name:         spec.Name,
		RefreshTimer: spec.RefreshTimer,
		Source:       spec.Source,
	}
}

func (app *Application) Run() {
	log.Info("Running Application", "name", app.Name)

	refreshTime, err := time.ParseDuration(app.RefreshTimer)
	if err != nil {
		app.Health = Suspended
		log.Error("Failed to parse refresh_time, it must be like \"3m30s\"", "name", app.Name)
		return
	}
	log.Info("Staring sync process")

	ticker := time.NewTicker(refreshTime)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		targetState, err := app.GetState()
		if err != nil {
			log.Warn("Not able to get service", "repo", app.Source.RepoURL)
			app.Health = Suspended
			continue
		}
		log.Info("got target state")
		if app.SyncStatus(targetState) {
			// TODO: Sync Status = Synched
			log.Info("Synched")
			app.Health = Healthy
			continue
		}
		log.Info("liveState and Target state is out of sync. syncing now...")

		// // TODO: Sync Status = Out of Sync
		app.Health = Progressing
		if err := app.Apply(targetState); err != nil {
			app.Health = Suspended
			log.Warn("Not able to targetState", "error", err.Error())
			continue
		}
		app.Health = Healthy
		log.Info("Applied new changes")
	}
}

func (app *Application) GetState() (string, error) {
	log.Info("Getting service state from git repo", "repo", app.Source.RepoURL, "app_name", app.Name)
	// TODO: not using targetRevision

	// TODO: IMPROVEMENT
	// Use Docker Volumes to clone repository
	// and then only fetch & pull if already exists
	// and check if specified path is modified then apply the changes
	fs := memfs.New()
	storage := memory.NewStorage()
	// defer clear storage, i (kunal singh) think that when storage goes out-of-scope
	// it is cleared

	_, err := git.Clone(storage, fs, &git.CloneOptions{
		URL: app.Source.RepoURL,
	})
	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		//  fetch & pull request
		// don't clone again
		log.Info("Repo already exits", "repo", app.Source.RepoURL)
		log.Error("Since the storage is not persistent, this error should not exist")
	} else if err != nil {
		return "", err
	}

	serviceFile, err := fs.Open(app.Source.Path)
	if err != nil {
		log.Error("Path not found", "repo", app.Source.RepoURL, "path", app.Source.Path)
		return "", err
	}
	defer serviceFile.Close()

	// reading the service file content
	buf := new(bytes.Buffer)
	buf.ReadFrom(serviceFile)

	return buf.String(), nil
}

func (app *Application) Apply(targetState string) error {
	log.Info("Applying new targetState")
	// TODO this client can be stored i app or new struct core
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Error("Not able to create a new docker client")
		return err
	}

	var swarmSpec spec.DockerSwarm
	yaml.Unmarshal([]byte(targetState), &swarmSpec)

	services, err := swarmSpec.GetServiceSpec(app.Name)
	log.Info("Get services from the source schema", "number of services found", len(services))

	for _, service := range services {
		res, err := cli.ServiceCreate(context.Background(), service, types.ServiceCreateOptions{})
		if err != nil {
			app.Health = Degraded
			log.Error("Not able to create a new service", "error", err.Error())
			return err
		}

		if len(res.Warnings) != 0 {
			log.Warn("New Service Create give warnings", "warnings", res.Warnings)

		}
	}

	app.LiveState = targetState
	return nil
}

// SyncStatus Check if LiveState = TargetState
//
// Whether or not the live state matches the target state.
// Is the deployed application the same as Git says it should be?
func (app *Application) SyncStatus(targetState string) bool {
	return app.LiveState == targetState
}

// Sync
// The process of making an application move to its target state.
// E.g. by applying changes to a docker swarm cluster.
func (app *Application) Sync(_ swarm.ServiceSpec) error {
	// TODO
	return nil
}
