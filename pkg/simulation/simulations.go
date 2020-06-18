package simulation

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/howardjohn/pilot-load/pkg/kube"
	"github.com/howardjohn/pilot-load/pkg/simulation/cluster"
	"github.com/howardjohn/pilot-load/pkg/simulation/model"
	"github.com/howardjohn/pilot-load/pkg/simulation/monitoring"
	"github.com/howardjohn/pilot-load/pkg/simulation/xds"
)

func Cluster(a model.Args) error {
	sim := cluster.NewNamespace(cluster.NamespaceSpec{
		Name:      "workload",
		Workloads: 2,
	})
	if err := ExecuteSimulations(a, sim); err != nil {
		return fmt.Errorf("error executing: %v", err)
	}
	return nil
}

func Adsc(a model.Args) error {
	return ExecuteSimulations(a, &xds.Simulation{
		Namespace: "default",
		Name:      "adsc",
		IP:        "1.2.3.4",
		// TODO: multicluster
		Cluster: "pilot-load",
	})
}

func ExecuteSimulations(a model.Args, simulation model.Simulation) error {
	cl, err := kube.NewClient(a.KubeConfig)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go captureTermination(ctx, cancel)
	defer cancel()
	go monitoring.StartMonitoring(ctx, 8765)
	simulationContext := model.Context{ctx, a, cl}
	if err := simulation.Run(simulationContext); err != nil {
		return err
	}
	<-ctx.Done()
	return simulation.Cleanup(simulationContext)
}

func captureTermination(ctx context.Context, cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
	}()
	select {
	case <-c:
		cancel()
	case <-ctx.Done():
	}
}
