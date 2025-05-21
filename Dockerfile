ARG PYTHON_VERSION=3.8
ARG RUST_VERSION=1.79.0

###
### Stage 0: python builder
###
FROM docker.io/python:${PYTHON_VERSION}-slim-bookworm as python-builder

# install the OS build deps

RUN apt-get update && apt-get install -y \
        build-essential \
        curl \
        libffi-dev \
        sqlite3 \
        libssl-dev \
        libjpeg-dev \
        libxslt1-dev \
        libxml2-dev

# for synapse rust dependencies
ARG RUST_VERSION
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain ${RUST_VERSION}
ENV PATH="/root/.cargo/bin:$PATH"

# install and build deps in a previous step so we do not need to do it on
# each change to the python code

RUN mkdir -p /synapse/synapse && touch /synapse/synapse/__init__.py
COPY synapse/pyproject.toml synapse/build_rust.py synapse/Cargo.* synapse/README.rst /synapse/
COPY synapse/rust /synapse/rust
RUN pip install --prefix="/install" --no-warn-script-location --no-clean \
        Twisted[tls]==24.7.0 \
        /synapse

# now install synapse itself to /install.

COPY synapse/ /synapse
RUN pip install --prefix="/install" --no-warn-script-location --no-deps \
        /synapse

###
### Stage 1: coap-proxy build
###

FROM docker.io/golang:1.24-bookworm as coap-proxy-builder
WORKDIR /build

COPY coap-proxy/go.mod coap-proxy/go.sum ./
RUN go mod download -x

COPY coap-proxy .
RUN go build -v

###
### Stage 2: meshmon build
###

FROM docker.io/golang:1.24-bookworm as meshmon-builder
WORKDIR /build

COPY meshmon/go.mod meshmon/go.sum ./
RUN go mod download -x

COPY meshmon .
RUN go build -v

###
### Stage 3: libksm build
###

FROM docker.io/debian:bookworm-slim as libksm-builder
WORKDIR /build

# for ksm_preload
RUN apt-get update && apt-get install -y \
        build-essential \
        git \
        cmake

# N.B. to work, this needs:
# echo 1 > /sys/kernel/mm/ksm/run
# echo 31250 > /sys/kernel/mm/ksm/pages_to_scan # 128MB of 4KB pages at a time
# echo 10000 > /sys/kernel/mm/ksm/pages_to_scan # 40MB of pages at a time
# ...to be run in the Docker host

RUN git clone https://github.com/unbrice/ksm_preload . && \
    cmake . && \
    make

###
### Stage 4.0: base runtime files
###

FROM docker.io/python:${PYTHON_VERSION}-slim-bookworm as synapse-runtime

RUN apt-get update && apt-get install -y sqlite3

COPY coap-proxy/maps /proxy/maps

COPY start-synapse.py /
COPY conf /conf

VOLUME ["/data"]

###
### Stage 4.1 minimal synapse image
###

FROM synapse-runtime as synapse

COPY --from=coap-proxy-builder /build/coap-proxy /proxy/bin/
COPY --from=python-builder /install /usr/local

EXPOSE 8008/tcp 8448/tcp 5683/udp

ENTRYPOINT ["/start-synapse.py"]

###
### Stage 4.2: synapse image for meshsim
###

FROM synapse-runtime as synapse-meshsim

# Install supervisord & other useful tools
RUN apt-get update && apt-get install -y \
    procps \
    net-tools \
    iproute2 \
    tcpdump \
    traceroute \
    mtr-tiny \
    inetutils-ping \
    less \
    lsof \
    supervisor \
    netcat-openbsd

# Install mateus' exp0
COPY mateus-exp0/requirements.txt mateus-exp0/requirements.txt
RUN python3 -m pip install -r mateus-exp0/requirements.txt
COPY mateus-exp0/*.py mateus-exp0

# Include prometheus node exporter
COPY --from=docker.io/prom/node-exporter /bin/node_exporter /bin/node_exporter

# Include meshsim's topologiser
COPY --from=gitlab.lip6.fr:5050/ie6/meshsim/topologiser:latest /bin/topologiser /topologiser

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

COPY --from=coap-proxy-builder /build/coap-proxy /proxy/bin/
COPY --from=python-builder /install /usr/local
COPY --from=libksm-builder /build/libksm_preload.so /usr/local/lib/

ENV LD_PRELOAD=/usr/local/lib/libksm_preload.so

# default is 32768 (8 4KB pages)
ENV KSMP_MERGE_THRESHOLD=16384

ENTRYPOINT ["/usr/bin/supervisord"]

###
### Stage 5: meshmon
###

FROM synapse-meshsim as synapse-meshmon

COPY --from=meshmon-builder /build/meshmon /usr/local/bin/

COPY supervisord-meshmon.conf /etc/supervisor/conf.d/supervisord.conf

###
### Stage 6: meshmon-has_events
###

FROM synapse-meshmon as synapse-meshmon-has_events

ENV SYNAPSE_CHECK_HAS_EVENTS=1
