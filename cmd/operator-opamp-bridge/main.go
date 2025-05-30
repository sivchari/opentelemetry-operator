// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/agent"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/proxy"
)

func main() {
	l := config.GetLogger()

	cfg, configLoadErr := config.Load(l, os.Args)
	if configLoadErr != nil {
		l.Error(configLoadErr, "Unable to load configuration")
		os.Exit(1)
	}
	l.Info("Starting the Remote Configuration service")

	kubeClient, kubeErr := cfg.GetKubernetesClient()
	if kubeErr != nil {
		l.Error(kubeErr, "Couldn't create kubernetes client")
		os.Exit(1)
	}
	operatorClient := operator.NewClient(cfg.Name, l.WithName("operator-client"), kubeClient, cfg.GetComponentsAllowed())

	opampClient := cfg.CreateClient()
	opampProxy := proxy.NewOpAMPProxy(l.WithName("server"), cfg.ListenAddr)
	opampAgent := agent.NewAgent(l.WithName("agent"), operatorClient, cfg, opampClient, opampProxy)

	if err := opampAgent.Start(); err != nil {
		l.Error(err, "Cannot start OpAMP client")
		os.Exit(1)
	}
	if err := opampProxy.Start(); err != nil {
		l.Error(err, "failed to start OpAMP Server")
		os.Exit(1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	opampAgent.Shutdown()
	proxyStopErr := opampProxy.Stop(context.Background())
	if proxyStopErr != nil {
		l.Error(proxyStopErr, "failed to shutdown proxy server")
	}
}
