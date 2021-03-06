package config

import (
	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/cloud/pkg/controller/constants"
)

// UpdatePodStatusWorkers is the count of goroutines of update pod status
var UpdatePodStatusWorkers int

// UpdateNodeStatusWorkers is the count of goroutines of update node status
var UpdateNodeStatusWorkers int

// QueryConfigMapWorkers is the count of goroutines of query configmap
var QueryConfigMapWorkers int

// QuerySecretWorkers is the count of goroutines of query secret
var QuerySecretWorkers int

func init() {
	if psw, err := config.CONFIG.GetValue("update-pod-status-workers").ToInt(); err != nil {
		UpdatePodStatusWorkers = constants.DefaultUpdatePodStatusWorkers
	} else {
		UpdatePodStatusWorkers = psw
	}
	log.LOGGER.Infof("update pod status workers: %d", UpdatePodStatusWorkers)

	if nsw, err := config.CONFIG.GetValue("update-node-status-workers").ToInt(); err != nil {
		UpdateNodeStatusWorkers = constants.DefaultUpdateNodeStatusWorkers
	} else {
		UpdateNodeStatusWorkers = nsw
	}
	log.LOGGER.Infof("update node status workers: %d", UpdateNodeStatusWorkers)

	if qcw, err := config.CONFIG.GetValue("query-configmap-workers").ToInt(); err != nil {
		QueryConfigMapWorkers = constants.DefaultQueryConfigMapWorkers
	} else {
		QueryConfigMapWorkers = qcw
	}
	log.LOGGER.Infof("query config map workers: %d", QueryConfigMapWorkers)

	if qsw, err := config.CONFIG.GetValue("query-secret-workers").ToInt(); err != nil {
		QuerySecretWorkers = constants.DefaultQuerySecretWorkers
	} else {
		QuerySecretWorkers = qsw
	}
	log.LOGGER.Infof("query secret workers: %d", QuerySecretWorkers)
}
