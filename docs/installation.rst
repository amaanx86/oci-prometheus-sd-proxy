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

   - Generate an API key pair in OCI Console under **Identity > Users > API Keys**
   - Add the user to an OCI group and attach the following IAM policies to that group in each tenancy:

   .. code-block:: text

       Allow group <group-name> to read instances in tenancy
       Allow group <group-name> to read compartments in tenancy
       Allow group <group-name> to read vnic-attachments in tenancy
       Allow group <group-name> to read vnics in tenancy
       Allow group <group-name> to read tag-namespaces in tenancy

   Replace ``<group-name>`` with the OCI group containing your API key user. These five policies cover all API calls the proxy makes:

   .. list-table::
      :header-rows: 1
      :widths: 20 30 30

      * - OCI Service
        - API Call
        - Policy Required
      * - Identity
        - ``ListCompartments``
        - ``read compartments``
      * - Compute
        - ``ListInstances``
        - ``read instances``
      * - Compute
        - ``ListVnicAttachments``
        - ``read vnic-attachments``
      * - Networking
        - ``GetVnic``
        - ``read vnics``
      * - Tagging
        - Tag-based filtering
        - ``read tag-namespaces``

   .. note::

      Missing any of these policies will cause ``NotAuthorizedOrNotFound`` errors in the logs for the affected compartments. The proxy will continue serving other tenancies and retain stale targets for the affected ones.

3. **Prepare configuration file**:

   .. code-block:: bash

       cp config.yaml.example config.yaml
       # Edit with your OCI tenancy details
