#!/bin/bash

# MIT License
#
# Copyright (c) 2020 Dmitrii Ustiugov, Plamen Petrov and EASE lab
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# Install K8 Power Manager
git clone https://github.com/intel/kubernetes-power-manager $HOME/kubernetes-power-manager

# Install Docker
sudo apt-get update
sudo apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg-agent

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# Set up the necessary Namespace, Service Account, and RBAC rules for the Kubernetes Power Manager
kubectl apply -f $HOME/kubernetes-power-manager/config/rbac/namespace.yaml
kubectl apply -f $HOME/kubernetes-power-manager/config/rbac/rbac.yaml

# Generate the CRD templates, create the Custom Resource Definitions, and install the CRDs and Built Docker images locally
cd $HOME/kubernetes-power-manager
make

# Apply Power Manager Controller
kubectl apply -f $HOME/kubernetes-power-manager/config/manager/manager.yaml

# Apply PowerConfig -> create the power-node-agent DaemonSet that manages the Power Node Agent pods.
kubectl apply -f $HOME/vhive/scripts/power_manager/power_config.yaml

# Apply Profile. U can modify the spec in the shared-profile.yaml file
kubectl apply -f $HOME/vhive/scripts/power_manager/shared-profile.yaml

# Apply the shared PowerWorkload. All CPUs (except reservedCPUs specified in this yaml file) will be tuned to the specified frequency in shared-profile.yaml
kubectl apply -f $HOME/vhive/scripts/power_manager/shared-workload.yaml

kubectl get powerprofiles -n intel-power
kubectl get powerworkloads -n intel-power
