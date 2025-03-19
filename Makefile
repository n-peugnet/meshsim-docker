HOSTIDS := 0 1

.PHONY: all
all: synapse-meshsim synapse-arm64

.PHONY: synapse
synapse:
	docker build -t synapse --target synapse .

.PHONY: synapse-meshsim
synapse-meshsim: synapse
	docker build -t synapse-meshsim .

.PHONY: synapse-arm64
synapse-arm64:
	docker build -t synapse-arm64 --target synapse --platform linux/arm64 .

.PHONY: meshsim
meshsim: synapse-meshsim
	meshsim --start=./start_hs.sh 0

.PHONY: push
push: synapse-arm64
	docker save synapse-arm64 | xz -T16 > /tmp/synapse-arm64
	for i in $(HOSTIDS); do ssh synapse$$i docker load < /tmp/synapse-arm64 & done; wait
	rm /tmp/synapse-arm64

.PHONY: run
run:
	echo $(HOSTIDS) | tr ' ' '\n' | parallel --verbose --ungroup \
		"docker -H tcp://synapse{} run --rm --rm -e SYNAPSE_SERVER_NAME=synapse{} -e SYNAPSE_REPORT_STATS=no -p 0.0.0.0:8448:8448 synapse-arm64"
