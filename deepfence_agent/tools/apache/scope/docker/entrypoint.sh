#!/bin/bash

if [ ! -v DF_PROG_NAME ]; then
  echo "Environment variable DF_PROG_NAME not set. It has to be any string"
  exit 0
fi

# $1 = MODE
# topology | discovery

# shellcheck disable=SC2034
ARGS=("$@")

MODE=""
TOPOLOGY_IP=""
if [ $# -eq 0 ]; then
  MODE="topology"
else
  MODE="$1"
  TOPOLOGY_IP="$2"
fi
if [[ -z "${TOPOLOGY_IP// }" ]]; then
  TOPOLOGY_IP="localhost"
fi

PROBE_PROCESSES=${DF_ENABLE_PROCESS_REPORT:-"true"}
PROBE_CONNECTIONS=${DF_ENABLE_CONNECTIONS_REPORT:-"true"}

if [[ "$MODE" == "discovery" ]]; then
  #This is needed for ElasticSearch. Since we are a privileged container, we set
  #it here
  echo "Setting sysctl values to be used later"
  sysctl -w vm.max_map_count=262144
  sysctl -w fs.pipe-max-size=536870912
  sysctl -w fs.file-max=1048576
  sysctl -w fs.nr_open=1048576
  sysctl -w net.core.somaxconn=10240
  sysctl -w net.ipv4.tcp_mem="1048576 1048576 1048576"
  sysctl -w net.ipv4.tcp_max_syn_backlog=1024
  sysctl -w net.ipv4.ip_local_port_range="1024 65534"
  probe_log_level=${LOG_LEVEL:-info}
  exec -a deepfence-discovery /home/deepfence/deepfence_exe --mode=probe --probe-only --weave=false --probe.no-controls=true --probe.log.level="$probe_log_level" --probe.spy.interval=5s --probe.publish.interval=10s --probe.docker.interval=10s --probe.docker=true --probe.insecure=true --probe.processes="$PROBE_PROCESSES" --probe.endpoint.report="$PROBE_CONNECTIONS" "http://$TOPOLOGY_IP:8004"
elif [[ "$MODE" == "topology" ]]; then
  app_log_level=${LOG_LEVEL:-info}
  export DF_PROG_NAME="topology"
  exec -a deepfence-topology /home/deepfence/deepfence_exe --mode=app --weave=false --probe.docker=true --app.externalUI=true --app.log.level="$app_log_level"
elif [[ "$MODE" == "cluster-agent" ]]; then
  probe_log_level=${LOG_LEVEL:-info}
  exec -a deepfence-cluster-agent /home/deepfence/deepfence_exe --mode=probe --probe-only --probe.kubernetes.role=cluster --probe.log.level="$probe_log_level" --weave=false --probe.docker=false --probe.spy.interval=5s --probe.publish.interval=10s --probe.insecure=true --probe.token="$DEEPFENCE_KEY" --probe.processes="$PROBE_PROCESSES" --probe.endpoint.report="$PROBE_CONNECTIONS" "https://$TOPOLOGY_IP"
fi
