HOSTIDS := 0 1
LOCAL_TARGETS := synapse-meshmon-has_events synapse-meshmon synapse-meshsim synapse

# Image used for running meshsim
export IMAGE ?= synapse-meshsim

.PHONY: all
all: $(LOCAL_TARGETS) synapse-arm64

.PHONY: $(LOCAL_TARGETS)
$(LOCAL_TARGETS):
	docker build -t $@ --target $@ .

.PHONY: synapse-arm64
synapse-arm64:
	docker build -t synapse-arm64 --target synapse --platform linux/arm64 .

.PHONY: meshsim
meshsim: $(IMAGE)
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
