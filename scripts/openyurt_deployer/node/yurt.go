// Author: Haoyuan Ma <flyinghorse0510@zju.edu.cn>
package node

import (
	"strings"
	"time"

	"github.com/vhive-serverless/vhive/scripts/openyurt_deployer/logs"
	"github.com/vhive-serverless/vhive/scripts/openyurt_deployer/template"
)

func (node *Node) CheckYurtMasterEnvironment() {
	node.OnlyExecByMaster()
	// Check environment
	var err error
	logs.InfoPrintf("Checking system environment...\n")

	// Check Helm
	_, err = node.LookPath("helm")
	if err != nil {
		logs.WarnPrintf("Helm not found! Helm will be automatically installed!\n")
	} else {
		logs.SuccessPrintf("Helm found!\n")
		node.Configs.Yurt.HelmInstalled = true
	}

	// Check Kustomize
	_, err = node.LookPath("kustomize")
	if err != nil {
		logs.WarnPrintf("Kustomize not found! Kustomize will be automatically installed!\n")
	} else {
		logs.SuccessPrintf("Kustomize found!\n")
		node.Configs.Yurt.KustomizeInstalled = true
	}

	// Add OS-specific dependencies to installation lists
	switch node.Configs.System.CurrentOS {
	case "ubuntu":
		node.Configs.Yurt.Dependencies = "curl apt-transport-https ca-certificates build-essential git"
	case "rocky linux":
		node.Configs.Yurt.Dependencies = ""
	case "centos":
		node.Configs.Yurt.Dependencies = ""
	default:
		logs.FatalPrintf("Unsupported OS: %s\n", node.Configs.System.CurrentOS)
	}

	logs.SuccessPrintf("Finished checking system environment!\n")
}

// Initialize Openyurt on master node
func (node *Node) YurtMasterInit() {
	node.OnlyExecByMaster()
	// Initialize
	var err error
	node.CheckYurtMasterEnvironment()
	node.CreateTmpDir()
	defer node.CleanUpTmpDir()

	// Install dependencies
	logs.WaitPrintf("Installing dependencies")
	err = node.InstallPackages(node.Configs.Yurt.Dependencies)
	logs.CheckErrorWithTagAndMsg(err, "Failed to install dependencies!\n")

	// Treat master as cloud node
	if node.Configs.Yurt.MasterAsCloud {
		logs.WarnPrintf("Master node WILL also be treated as a cloud node!\n")
		node.ExecShellCmd("kubectl taint nodes --all node-role.kubernetes.io/master:NoSchedule-")
		node.ExecShellCmd("kubectl taint nodes --all node-role.kubernetes.io/control-plane-")
	}

	// Install helm
	if !node.Configs.Yurt.HelmInstalled {
		switch node.Configs.System.CurrentOS {
		case "ubuntu":
			// Download public signing key && Add the Helm apt repository
			logs.WaitPrintf("Downloading public signing key && Add the Helm apt repository")
			// Download public signing key
			filePathName, err := node.DownloadToTmpDir(node.Configs.Yurt.HelmPublicSigningKeyDownloadUrl)
			logs.CheckErrorWithMsg(err, "Failed to download public signing key && add the Helm apt repository!\n")
			_, err = node.ExecShellCmd("sudo mkdir -p /usr/share/keyrings && cat %s | gpg --dearmor | sudo tee /usr/share/keyrings/helm.gpg > /dev/null", filePathName)
			logs.CheckErrorWithMsg(err, "Failed to download public signing key && add the Helm apt repository!\n")
			// Add the Helm apt repository
			_, err = node.ExecShellCmd(`echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list`)
			logs.CheckErrorWithTagAndMsg(err, "Failed to download public signing key && add the Helm apt repository!\n")
			// Install helm
			logs.WaitPrintf("Installing Helm")
			err = node.InstallPackages("helm")
			logs.CheckErrorWithTagAndMsg(err, "Failed to install helm!\n")
		default:
			logs.FatalPrintf("Unsupported Linux distribution: %s\n", node.Configs.System.CurrentOS)
		}
	}

	// Install kustomize
	if !node.Configs.Yurt.KustomizeInstalled {
		// Download kustomize helper script
		logs.WaitPrintf("Downloading kustomize")
		filePathName, err := node.DownloadToTmpDir(node.Configs.Yurt.KustomizeScriptDownloadUrl)
		logs.CheckErrorWithMsg(err, "Failed to download kustomize!\n")
		// Download kustomize
		_, err = node.ExecShellCmd("chmod u+x %s && %s %s", filePathName, filePathName, node.Configs.System.TmpDir)
		logs.CheckErrorWithTagAndMsg(err, "Failed to download kustomize!\n")
		// Install kustomize
		logs.WaitPrintf("Installing kustomize")
		_, err = node.ExecShellCmd("sudo cp %s /usr/local/bin", node.Configs.System.TmpDir+"/kustomize")
		logs.CheckErrorWithTagAndMsg(err, "Failed to Install kustomize!\n")
	}

	// Add OpenYurt repo with helm
	logs.WaitPrintf("Adding OpenYurt repo(version %s) with helm", node.Configs.Yurt.YurtVersion)
	_, err = node.ExecShellCmd("git clone --quiet https://github.com/openyurtio/openyurt-helm.git %s/openyurt-helm && pushd %s/openyurt-helm && git checkout openyurt-%s && popd", node.Configs.System.TmpDir, node.Configs.System.TmpDir, node.Configs.Yurt.YurtVersion)
	logs.CheckErrorWithTagAndMsg(err, "Failed to add OpenYurt repo with helm!\n")

	// Deploy yurt-app-manager
	logs.WaitPrintf("Deploying yurt-app-manager")
	_, err = node.ExecShellCmd("helm install yurt-app-manager -n kube-system %s/openyurt-helm/charts/yurt-app-manager", node.Configs.System.TmpDir)
	logs.CheckErrorWithTagAndMsg(err, "Failed to deploy yurt-app-manager!\n")

	// Wait for yurt-app-manager to be ready
	logs.WaitPrintf("Waiting for yurt-app-manager to be ready")
	waitCount := 1
	for {
		yurtAppManagerStatus, err := node.ExecShellCmd(`kubectl get pod -n kube-system | grep yurt-app-manager | sed -n "s/\s*\(\S*\)\s*\(\S*\)\s*\(\S*\).*/\2 \3/p"`)
		logs.CheckErrorWithMsg(err, "Failed to wait for yurt-app-manager to be ready!\n")
		if yurtAppManagerStatus == "1/1 Running" {
			logs.SuccessPrintf("\n")
			break
		} else {
			logs.WarnPrintf("Waiting for yurt-app-manager to be ready [%ds]\n", waitCount)
			waitCount += 1
			time.Sleep(time.Second)
		}
	}

	// Deploy yurt-controller-manager
	logs.WaitPrintf("Deploying yurt-controller-manager")
	_, err = node.ExecShellCmd("helm install openyurt %s/openyurt-helm/charts/openyurt -n kube-system", node.Configs.System.TmpDir)
	logs.CheckErrorWithTagAndMsg(err, "Failed to deploy yurt-controller-manager!\n")

	// Setup raven-controller-manager Component
	// Clone repository
	logs.WaitPrintf("Cloning repo: raven-controller-manager")
	_, err = node.ExecShellCmd("git clone --quiet https://github.com/openyurtio/raven-controller-manager.git %s/raven-controller-manager", node.Configs.System.TmpDir)
	logs.CheckErrorWithTagAndMsg(err, "Failed to clone repo: raven-controller-manager!\n")
	// Deploy raven-controller-manager
	logs.WaitPrintf("Deploying raven-controller-manager")
	_, err = node.ExecShellCmd("pushd %s/raven-controller-manager && git checkout v0.3.0 && make generate-deploy-yaml && kubectl apply -f _output/yamls/raven-controller-manager.yaml && popd", node.Configs.System.TmpDir)
	logs.CheckErrorWithTagAndMsg(err, "Failed to deploy raven-controller-manager!\n")

	// Setup raven-agent Component
	// Clone repository
	logs.WaitPrintf("Cloning repo: raven-agent")
	_, err = node.ExecShellCmd("git clone --quiet https://github.com/openyurtio/raven.git %s/raven-agent", node.Configs.System.TmpDir)
	logs.CheckErrorWithTagAndMsg(err, "Failed to clone repo: raven-agent!\n")
	// Deploy raven-agent
	logs.WaitPrintf("Deploying raven-agent")
	_, err = node.ExecShellCmd("pushd %s/raven-agent && git checkout v0.3.0 && FORWARD_NODE_IP=true make deploy && popd", node.Configs.System.TmpDir)
	logs.CheckErrorWithTagAndMsg(err, "Failed to deploy raven-agent!\n")
}

// Expand Openyurt to worker node
func (node *Node) YurtMasterExpand(worker *Node) {
	node.OnlyExecByMaster()
	// Initialize
	var err error
	var workerAsEdge string

	// Label worker node as cloud/edge
	logs.WaitPrintf("Labeling worker node: %s", worker.Configs.System.NodeHostName)
	if worker.NodeRole == "edge" {
		workerAsEdge = "true"
	} else if worker.NodeRole == "cloud" {
		workerAsEdge = "false"
	} else {
		logs.FatalPrintf("worker's role must be edge or cloud, but this node's role is %s", worker.NodeRole)
	}
	_, err = node.ExecShellCmd("kubectl label node %s openyurt.io/is-edge-worker=%s --overwrite", worker.Configs.System.NodeHostName, workerAsEdge)
	logs.CheckErrorWithTagAndMsg(err, "Failed to label worker node!\n")

	// Activate the node autonomous mode
	logs.WaitPrintf("Activating the node autonomous mode")
	_, err = node.ExecShellCmd("kubectl annotate node %s node.beta.openyurt.io/autonomy=true --overwrite", worker.Configs.System.NodeHostName)
	logs.CheckErrorWithTagAndMsg(err, "Failed to activate the node autonomous mode!\n")

	// Wait for worker node to be Ready
	logs.WaitPrintf("Waiting for worker node to be ready")
	waitCount := 1
	for {
		workerNodeStatus, err := node.ExecShellCmd(`kubectl get nodes | sed -n "/.*%s.*/p" | sed -n "s/\s*\(\S*\)\s*\(\S*\).*/\2/p"`, worker.Configs.System.NodeHostName)
		logs.CheckErrorWithMsg(err, "Failed to wait for worker node to be ready!\n")
		if workerNodeStatus == "Ready" {
			logs.SuccessPrintf("\n")
			break
		} else {
			logs.WarnPrintf("Waiting for worker node to be ready [%ds]\n", waitCount)
			waitCount += 1
			time.Sleep(time.Second)
		}
	}

	// Restart pods in the worker node
	logs.WaitPrintf("Restarting pods in the worker node")
	shellOutput, err := node.ExecShellCmd(template.GetRestartPodsShell(), worker.Configs.System.NodeHostName)
	logs.CheckErrorWithMsg(err, "Failed to restart pods in the worker node!\n")
	podsToBeRestarted := strings.Split(shellOutput, "\n")
	for _, pods := range podsToBeRestarted {
		podsInfo := strings.Split(pods, " ")
		logs.WaitPrintf("Restarting pod: %s => %s", podsInfo[0], podsInfo[1])
		_, err = node.ExecShellCmd("kubectl -n %s delete pod %s", podsInfo[0], podsInfo[1])
		logs.CheckErrorWithTagAndMsg(err, "Failed to restart pods in the worker node!\n")
	}
}

// Join existing Kubernetes worker node to Openyurt cluster
func (node *Node) YurtWorkerJoin(addr string, port string, token string) {

	// Initialize
	var err error

	// Set up Yurthub
	logs.WaitPrintf("Setting up Yurthub")
	_, err = node.ExecShellCmd(
		"echo '%s' | sed -e 's|__kubernetes_master_address__|%s:%s|' -e 's|__bootstrap_token__|%s|' | sudo tee /etc/kubernetes/manifests/yurthub-ack.yaml",
		template.GetYurtHubConfig(),
		addr,
		port,
		token)
	logs.CheckErrorWithTagAndMsg(err, "Failed to set up Yurthub!\n")

	// Configure Kubelet
	logs.WaitPrintf("Configuring kubelet")
	node.ExecShellCmd("sudo mkdir -p /var/lib/openyurt && echo '%s' | sudo tee /var/lib/openyurt/kubelet.conf", template.GetKubeletConfig())
	logs.CheckErrorWithMsg(err, "Failed to configure kubelet!\n")
	node.ExecShellCmd(`sudo sed -i "s|KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=\/etc\/kubernetes\/bootstrap-kubelet.conf\ --kubeconfig=\/etc\/kubernetes\/kubelet.conf|KUBELET_KUBECONFIG_ARGS=--kubeconfig=\/var\/lib\/openyurt\/kubelet.conf|g" /etc/systemd/system/kubelet.service.d/10-kubeadm.conf`)
	logs.CheckErrorWithMsg(err, "Failed to configure kubelet!\n")
	node.ExecShellCmd("sudo systemctl daemon-reload && sudo systemctl restart kubelet")
	logs.CheckErrorWithTagAndMsg(err, "Failed to configure kubelet!\n")
}

func (node *Node) YurtWorkerClean(){
	node.OnlyExecByWorker()
	var err error
	logs.WaitPrintf("Cleaning openyurt kubelet on node:%s", node.Name)
	_, err = node.ExecShellCmd("sudo rm -rf /var/lib/openyurt")
	_, err = node.ExecShellCmd("sudo rm /etc/kubernetes/pki/ca.crt")
	_, err = node.ExecShellCmd(`sudo sed -i "s|KUBELET_KUBECONFIG_ARGS=--kubeconfig=\/var\/lib\/openyurt\/kubelet.conf|KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=\/etc\/kubernetes\/bootstrap-kubelet.conf\ --kubeconfig=\/etc\/kubernetes\/kubelet.conf|g" /etc/systemd/system/kubelet.service.d/10-kubeadm.conf`)
	logs.CheckErrorWithMsg(err, "Failed to clean kubelet on node: %s", node.Name)
}

