# Synapse for Meshsim

Docker image to run Synapse in [Meshsim](https://gitlab.lip6.fr/ie6/meshsim).
A user `matthew` with passord `secret` and a default token `fake_token` is created on each node,
to simplify making Matrix API call.

## Build the image

 * Install Docker (needs API 1.38; API 1.2x is known not to work.).
   * On Debian and derivatives: `sudo apt install docker.io`
   * Check the API with `docker version`.

 * Optional: Enable KSM on your host so your synapses can deduplicate RAM
   as much as possible

   ```sh
   screen ~/Library/Containers/com.docker.docker/Data/vms/0/tty  # on Docker-for-Mac
   echo 1 > /sys/kernel/mm/ksm/run
   echo 10000 > /sys/kernel/mm/ksm/pages_to_scan # 40MB of pages at a time

   # check to see if it's working (will only kick in once you start running something which requests KSM, like our KSMified synapse)
   grep -H '' /sys/kernel/mm/ksm/run/*
   ```

 * Build the (KSM-capable) docker image:

   * Clone the `synapse-meshsim` repo with its submodules:
     ```
     git clone --recurse-submodules https://gitlab.lip6.fr/ie6/synapse-meshsim
     cd synapse-meshsim
     ```
     Or, if it is already cloned, update the submodules:
     ```
     git submodule init
     git submodule update
     ```

   * Run `docker build -t synapse .`

#### Usage with Meshsim

Install [Meshsim](https://gitlab.lip6.fr/ie6/meshsim),
then run `meshsim --start=./start_hs.sh 0`.

 * Optionally edit `start_hs.sh` to add bind mount to a local working copy of
   synapse. This allows doing synapse dev without having to rebuild images. See
   `start_hs.sh` for details. An example of the `docker run` command in `start_hs.sh` is below:

   ```
   docker run -d --name node$NETWORK_ID.$HSID \
   	--privileged \
   	--network mesh$NETWORK_ID \
   	--hostname node$HSID \
   	-e SYNAPSE_SERVER_NAME=node$HSID \
   	-e SYNAPSE_REPORT_STATS=no \
   	-e SYNAPSE_ENABLE_REGISTRATION=yes \
   	-e SYNAPSE_LOG_LEVEL=INFO \
   	-p $((18000 + HSID + NETWORK_ID * 100)):8008 \
   	-p $((19000 + HSID + NETWORK_ID * 100)):3000 \
   	-p $((20000 + HSID + NETWORK_ID * 100)):5683/udp \
   	-e SYNAPSE_LOG_HOST=$HOST_IP:$((3000 + NETWORK_ID * 100)) \
   	-e PROXY_DUMP_PAYLOADS=1 \
   	--mount type=bind,source=/home/user/matrix-low-bandwidth/coap-proxy,destination=/proxy \
   	--mount type=bind,source=/home/user/matrix-low-bandwidth/synapse/synapse,destination=/usr/local/lib/python3.7/site-packages/synapse \
   	synapse
   ```

#### Step-by-step checks


To verify that everythig is working, you can realise the following checks.

 * create a docker network: `docker network create --driver bridge mesh0`. Later
   we will need to know the gateway IP (so that the images can talk to
   meshsim, etc on the host). On MacOS `host.docker.internal` will work,
   otherwise run `docker network inspect mesh` and find the Gateway IP.
 * check you can start a synapse via `./start_hs.sh 0 1 $DOCKER_IP` with 0 as networkid, 1 as hsid and DOCKER_IP being the docker network gateway IP.
 * check if it's running with `docker stats`
 * check the supervisor logs with `docker logs -f node0.1`
 * log into the container to poke around with `docker exec -it node0.1 /bin/bash`
    * Actual synapse logs are located at `/var/log/supervisor/synapse*`

 * Check you can connect to its synapse at http://localhost:18001 (ports are 18000 + hsid + networkid*100).
   * Requires a Riot running on http on localhost or similar to support CORS to non-https
   * Initial user sign up may time out due to trying to connect to Riot-bot. Simply refresh the page and you should get in fine.
   * The KSM'd dockerfile autoprovisions an account on the HS called l/p matthew/secret for testing purposes.
 * Check that the topologiser is listening at http://localhost:19001 (ports are 19000 + hsid + networkid*100)
    * Don't expect to navigate to this URL and see anything more than a 404. As long as *something* is listening at this port, things are set up correctly.

 * shut it down nicely:

       docker stop node0.1
       docker rm node0.1
       docker network rm mesh0

#### Using the CoAP proxy

* Build the proxy (see instruction in the [proxy's README](https://github.com/matrix-org/coap-proxy/blob/master/README.md))
* Run it by telling it to talk to the HS's proxy:

  ```bash
  ./bin/coap-proxy --coap-target localhost:20001 # Ports are 20000 + hsid
  ```

* Make clients talk to http://localhost:8888
* => profit

