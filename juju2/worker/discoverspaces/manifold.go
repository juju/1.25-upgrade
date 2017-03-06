// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package discoverspaces

import (
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/api/base"
	"github.com/juju/1.25-upgrade/juju2/api/discoverspaces"
	"github.com/juju/1.25-upgrade/juju2/environs"
	"github.com/juju/1.25-upgrade/juju2/network"
	"github.com/juju/1.25-upgrade/juju2/worker"
	"github.com/juju/1.25-upgrade/juju2/worker/dependency"
	"github.com/juju/1.25-upgrade/juju2/worker/gate"
)

type ManifoldConfig struct {
	APICallerName string
	EnvironName   string
	UnlockerName  string

	NewFacade func(base.APICaller) (Facade, error)
	NewWorker func(Config) (worker.Worker, error)
}

func NewFacade(apiCaller base.APICaller) (Facade, error) {
	return discoverspaces.NewAPI(apiCaller), nil
}

func Manifold(config ManifoldConfig) dependency.Manifold {
	inputs := []string{config.APICallerName, config.EnvironName}
	if config.UnlockerName != "" {
		inputs = append(inputs, config.UnlockerName)
	}
	return dependency.Manifold{
		Inputs: inputs,
		Start:  startFunc(config),
	}
}

func startFunc(config ManifoldConfig) dependency.StartFunc {
	return func(context dependency.Context) (worker.Worker, error) {

		// optional unlocker, might stay nil
		var unlocker gate.Unlocker
		if config.UnlockerName != "" {
			if err := context.Get(config.UnlockerName, &unlocker); err != nil {
				return nil, errors.Trace(err)
			}
		}

		var environ environs.Environ
		if err := context.Get(config.EnvironName, &environ); err != nil {
			return nil, errors.Trace(err)
		}

		var apiCaller base.APICaller
		if err := context.Get(config.APICallerName, &apiCaller); err != nil {
			return nil, errors.Trace(err)
		}
		facade, err := config.NewFacade(apiCaller)
		if err != nil {
			return nil, errors.Trace(err)
		}

		w, err := config.NewWorker(Config{
			Facade:   facade,
			Environ:  environ,
			NewName:  network.ConvertSpaceName,
			Unlocker: unlocker,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		return w, nil
	}
}
