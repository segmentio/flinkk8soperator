#!/usr/bin/env sh

set -e

make docker_build
docker save flinkk8soperator > flinkk8soperator.tar

sudo snap install microk8s --classic --channel=1.12/stable

sh boilerplate/lyft/golang_test_targets/dep_install.sh

dep ensure
