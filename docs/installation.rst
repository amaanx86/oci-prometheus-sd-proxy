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

2. **Grant OCI permissions** - choose one of two authentication methods:

   .. rubric:: Option A: API key auth (works anywhere)

   Generate an API key pair in OCI Console under **Identity > Users > API Keys**.
   Add the user to an OCI group and attach the following IAM policies to that group in each tenancy:

   .. code-block:: text

       Allow group <group-name> to read instances in tenancy
       Allow group <group-name> to read compartments in tenancy
       Allow group <group-name> to read vnic-attachments in tenancy
       Allow group <group-name> to read vnics in tenancy
       Allow group <group-name> to read tag-namespaces in tenancy

   Replace ``<group-name>`` with the OCI group containing your API key user.

   .. rubric:: Option B: Instance principal auth (proxy runs on OCI compute)

   No API key is needed. The compute instance authenticates using its own identity via OCI IMDS.

   **Step 1** - Create a dynamic group that matches the instance running the proxy.
   In OCI Console go to **Identity > Dynamic Groups > Create Dynamic Group**:

   .. code-block:: text

       All {instance.id = '<your-instance-ocid>'}

   Or to match any instance in a compartment:

   .. code-block:: text

       All {instance.compartment.id = '<compartment-ocid>'}

   **Step 2** - Create IAM policies granting the dynamic group read access.
   In OCI Console go to **Identity > Policies > Create Policy** at the tenancy (root) level:

   .. code-block:: text

       Allow dynamic-group <dynamic-group-name> to read instances in tenancy
       Allow dynamic-group <dynamic-group-name> to read compartments in tenancy
       Allow dynamic-group <dynamic-group-name> to read vnic-attachments in tenancy
       Allow dynamic-group <dynamic-group-name> to read vnics in tenancy
       Allow dynamic-group <dynamic-group-name> to read tag-namespaces in tenancy

   Replace ``<dynamic-group-name>`` with the dynamic group created in Step 1.

   .. note::

      Policies for instance principals must be created at the **tenancy (root) level**, not at the compartment level, for ``ListCompartments`` to traverse the full compartment tree.

   **Step 3** - Set ``auth_type: instance_principal`` in ``config.yaml`` (see :doc:`configuration`).

   ---

   These five permissions cover all API calls the proxy makes regardless of auth method:

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
