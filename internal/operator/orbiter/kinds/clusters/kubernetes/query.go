package kubernetes

import (
	"github.com/caos/orbos/internal/api"
	"github.com/caos/orbos/internal/git"
	"github.com/pkg/errors"

	"github.com/caos/orbos/internal/operator/common"
	"github.com/caos/orbos/internal/operator/orbiter"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/core/infra"
	"github.com/caos/orbos/mntr"
)

func query(
	monitor mntr.Monitor,
	clusterID string,
	desired *DesiredV0,
	current *CurrentCluster,
	providerCurrents map[string]interface{},
	nodeAgentsCurrent *common.CurrentNodeAgents,
	nodeAgentsDesired *common.DesiredNodeAgents,
	k8sClient *Client,
	oneoff bool,
	gitClient *git.Client,
) (orbiter.EnsureFunc, error) {

	cloudPools, kubeAPIAddress, err := GetProviderInfos(desired, providerCurrents)
	if err != nil {
		panic(err)
	}

	if err := poolIsConfigured(&desired.Spec.ControlPlane, cloudPools); err != nil {
		return nil, err
	}

	for _, w := range desired.Spec.Workers {
		if err := poolIsConfigured(w, cloudPools); err != nil {
			return nil, err
		}
	}

	controlplane, controlplaneMachines, workers, workerMachines, initializeMachine, uninitializeMachine, err := initialize(
		monitor,
		current,
		*desired,
		nodeAgentsCurrent,
		nodeAgentsDesired,
		cloudPools,
		k8sClient,
		func(machine *initializedMachine) {
			firewallFunc(monitor, *desired)(machine)
		})

	return func(psf api.PushDesiredFunc) *orbiter.EnsureResult {
		return orbiter.ToEnsureResult(ensure(
			monitor,
			clusterID,
			desired,
			kubeAPIAddress,
			psf,
			k8sClient,
			oneoff,
			controlplane,
			controlplaneMachines,
			workers,
			workerMachines,
			initializeMachine,
			uninitializeMachine,
			gitClient,
		))
	}, err
}

func poolIsConfigured(poolSpec *Pool, infra map[string]map[string]infra.Pool) error {
	prov, ok := infra[poolSpec.Provider]
	if !ok {
		return errors.Errorf("provider %s not configured", poolSpec.Provider)
	}
	if _, ok := prov[poolSpec.Pool]; !ok {
		return errors.Errorf("pool %s not configured on provider %s", poolSpec.Provider, poolSpec.Pool)
	}
	return nil
}

func GetProviderInfos(desired *DesiredV0, providerCurrents map[string]interface{}) (map[string]map[string]infra.Pool, *infra.Address, error) {
	cloudPools := make(map[string]map[string]infra.Pool)
	var kubeAPIAddress *infra.Address

	for providerName, provider := range providerCurrents {
		if cloudPools[providerName] == nil {
			cloudPools[providerName] = make(map[string]infra.Pool)
		}
		prov := provider.(infra.ProviderCurrent)
		providerPools := prov.Pools()
		providerIngresses := prov.Ingresses()
		for providerPoolName, providerPool := range providerPools {
			cloudPools[providerName][providerPoolName] = providerPool
			if desired.Spec.ControlPlane.Provider == providerName && desired.Spec.ControlPlane.Pool == providerPoolName {
				var ok bool
				kubeAPIAddress, ok = providerIngresses["kubeapi"]
				if !ok {
					return nil, nil, errors.New("no externally reachable address named kubeapi found")
				}
			}
		}
	}
	return cloudPools, kubeAPIAddress, nil
}
