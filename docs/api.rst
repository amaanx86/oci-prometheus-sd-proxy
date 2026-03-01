API Reference
==============

Endpoints
---------

GET /v1/targets
~~~~~~~~~~~~~~~

Prometheus HTTP Service Discovery endpoint.

**Authentication**: Bearer token required

**Response**: JSON array of target groups

.. code-block:: bash

    curl -H "Authorization: Bearer YOUR_TOKEN" \
      http://localhost:8080/v1/targets

Response format:

.. code-block:: json

    [
      {
        "targets": ["10.0.1.5:9100"],
        "labels": {
          "__meta_oci_instance_name": "instance-name",
          "__meta_oci_tenancy_name": "tenancy-name",
          "__meta_oci_region": "me-jeddah-1"
        }
      }
    ]

GET /healthz
~~~~~~~~~~~~

Liveness probe (health check).

**Authentication**: Not required

**Response**: Plain text "ok"

.. code-block:: bash

    curl http://localhost:8080/healthz

GET /readyz
~~~~~~~~~~~

Readiness probe.

**Authentication**: Not required

**Response**: Plain text "ok"

.. code-block:: bash

    curl http://localhost:8080/readyz
