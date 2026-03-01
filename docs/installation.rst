Installation
=============

Docker (Recommended)
--------------------

Pull and run from GitHub Container Registry:

.. code-block:: bash

    docker run -d \
      -e SERVER_TOKEN="$(openssl rand -hex 32)" \
      -v /path/to/config.yaml:/etc/oci-sd/config.yaml:ro \
      -v /path/to/oci/keys:/etc/oci-sd/keys:ro \
      -p 8080:8080 \
      ghcr.io/amaanx86/oci-prometheus-sd-proxy:latest

Docker Compose
--------------

Simplest setup using Docker Compose:

.. code-block:: bash

    cd deploy/docker
    cp .env.example .env
    cp config.yaml.example config.yaml
    mkdir -p oci-keys
    cp ~/.oci/api_key.pem oci-keys/

    # Edit .env and set SERVER_TOKEN
    openssl rand -hex 32

    docker-compose -f docker-compose-production.yml up -d

Binary
------

Build from source:

.. code-block:: bash

    git clone https://github.com/amaanx86/oci-prometheus-sd-proxy.git
    cd oci-prometheus-sd-proxy
    make build

    # Binary at ./bin/oci-sd-proxy
    ./bin/oci-sd-proxy

OCI Prerequisites
-----------------

Before running the service:

1. **Tag your instances** in OCI Console with the monitoring tag:

   - Key: ``monitoring``
   - Value: ``enabled``

2. **Create API credentials** for each OCI tenancy:

   - Generate API key pair
   - User must have permissions:
     - ``inspect instance-family`` on all compartments
     - ``inspect virtual-network-family`` on all compartments

3. **Prepare configuration file**:

   .. code-block:: bash

       cp config.yaml.example config.yaml
       # Edit with your OCI tenancy details
