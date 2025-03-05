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
