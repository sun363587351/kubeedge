/*
Copyright 2019 The KubeEdge Authors.

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

package common

import (
	"net/http"
	"os/exec"
	"time"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"strconv"
)

//K8s resource handlers
const (
	AppHandler        = "/api/v1/namespaces/default/pods"
	NodeHandler       = "/api/v1/nodes"
	DeploymentHandler = "/apis/apps/v1/namespaces/default/deployments"
	ConfigmapHandler  = "/api/v1/namespaces/default/configmaps"
	ServiceHandler    = "/api/v1/namespaces/default/services"
)
var(
	chconfigmapRet = make(chan error)
	Deployments []string
	NodeInfo = make(map[string][]string)
	CloudConfigMap string
	CloudCoreDeployment string
	ToTaint bool
	EdgeNode string
)

func HandleCloudDeployment(cloudConfigMap, cloudCoreDeployment, apiserver2, confighdl, deploymenthdl, imgURL string, nodelimit int) error{
	nodes := strconv.FormatInt(int64(nodelimit), 10)
	cmd := exec.Command("bash", "-x", "scripts/update_configmap.sh", "create_cloud_config", "", apiserver2, cloudConfigMap, nodes)
	err := utils.PrintCombinedOutput(cmd)
	Expect(err).Should(BeNil())
	go utils.HandleConfigmap(chconfigmapRet, http.MethodPost, confighdl, false)
	ret := <-chconfigmapRet
	Expect(ret).To(BeNil())

	//Handle cloudCore deployment
	go utils.HandleDeployment(true, false, http.MethodPost, deploymenthdl, cloudCoreDeployment, imgURL, "", cloudConfigMap, 1)

	return nil
}

func HandleEdgeDeployment(cloudhub, depHandler, nodeHandler, cmHandler, imgURL, podHandler string, numOfNodes int ) v1.PodList {
	replica := 1
	//Create edgecore configMaps based on the users choice of edgecore deployment.
	for i := 0; i < numOfNodes; i++ {
		nodeName := "perf-node-" + utils.GetRandomString(10)
		nodeSelector := "node-" + utils.GetRandomString(3)
		configmap := "edgecore-configmap-" + utils.GetRandomString(3)
		//Register EdgeNodes to K8s Master
		go utils.RegisterNodeToMaster(nodeName, nodeHandler, nodeSelector)
		cmd := exec.Command("bash", "-x", "scripts/update_configmap.sh", "create_edge_config", nodeName, cloudhub, configmap)
		err := utils.PrintCombinedOutput(cmd)
		Expect(err).Should(BeNil())
		//Create ConfigMaps for Each EdgeNode created
		go utils.HandleConfigmap(chconfigmapRet, http.MethodPost, cmHandler, true)
		ret := <-chconfigmapRet
		Expect(ret).To(BeNil())
		//Store the ConfigMap against each edgenode
		NodeInfo[nodeName] = append(NodeInfo[nodeName], configmap)
	}
	//Create edgeCore deployments as users configuration
	for _, configmap := range NodeInfo {
		UID := "edgecore-deployment-" + utils.GetRandomString(5)
		go utils.HandleDeployment(false, true, http.MethodPost, depHandler, UID, imgURL, "", configmap[0], replica)
		Deployments = append(Deployments, UID)
	}
	time.Sleep(2 * time.Second)
	podlist, err := utils.GetPods(podHandler, "")
	Expect(err).To(BeNil())
	utils.CheckPodRunningState(podHandler, podlist)

	//Check All EdgeNode are in Running state
	Eventually(func() int {
		count := 0
		for edgenodeName, _ := range NodeInfo {
			status := utils.CheckNodeReadyStatus(nodeHandler, edgenodeName)
			utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
			if status == "Running" {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(numOfNodes), "Nodes register to the k8s master is unsuccessfull !!")

	return podlist
}

func DeleteEdgeDeployments(apiServer string, nodes int){
	//delete confogMap
	for _, configmap := range NodeInfo {
		go utils.HandleConfigmap(chconfigmapRet, http.MethodDelete, apiServer+ConfigmapHandler+"/"+configmap[0], false)
		ret := <-chconfigmapRet
		Expect(ret).To(BeNil())

	}
	//delete edgenode deployment
	for _, depName := range Deployments {
		go utils.HandleDeployment(true, true, http.MethodDelete, apiServer+DeploymentHandler+"/"+depName, "", "", "", "", 0)
	}
	//delete edgenodes
	for edgenodeName, _ := range NodeInfo {
		err := utils.DeRegisterNodeFromMaster(apiServer+NodeHandler, edgenodeName)
		if err != nil {
			utils.Failf("DeRegisterNodeFromMaster failed: %v", err)
		}
	}
	//Verify deployments, configmaps, nodes are deleted successfully
	Eventually(func() int {
		count := 0
		for _, depName := range Deployments {
			statusCode := utils.VerifyDeleteDeployment(apiServer + DeploymentHandler + "/" + depName)
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(len(Deployments)), "EdgeNode deployments delete unsuccessfull !!")

	Eventually(func() int {
		count := 0
		for _, configmap := range NodeInfo {
			statusCode := utils.GetConfigmap(apiServer + ConfigmapHandler + "/" + configmap[0])
			if statusCode == 404 {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(len(Deployments)), "EdgeNode configMaps delete unsuccessfull !!")

	Eventually(func() int {
		count := 0
		for edgenodeName, _ := range NodeInfo {
			status := utils.CheckNodeDeleteStatus(apiServer+NodeHandler, edgenodeName)
			utils.Info("Node Name: %v, Node Status: %v", edgenodeName, status)
			if status == 404 {
				count++
			}
		}
		return count
	}, "60s", "4s").Should(Equal(nodes), "EdgeNode deleton is unsuccessfull !!")

	NodeInfo = nil

}

func DeleteCloudDeployment(apiserver string){
	//delete cloud deployment
	go utils.HandleDeployment(true, true, http.MethodDelete, apiserver+DeploymentHandler+"/"+CloudCoreDeployment, "", "", "", "", 0)
	//delete cloud configMap
	go utils.HandleConfigmap(chconfigmapRet, http.MethodDelete, apiserver+ConfigmapHandler+"/"+CloudConfigMap, false)
	ret := <-chconfigmapRet
	Expect(ret).To(BeNil())
	//delete cloud svc
	StatusCode := utils.DeleteSvc(apiserver + ServiceHandler + "/" + CloudCoreDeployment)
	Expect(StatusCode).Should(Equal(http.StatusOK))
}