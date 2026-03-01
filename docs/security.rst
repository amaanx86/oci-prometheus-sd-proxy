Security
=========

Authentication
--------------

All API endpoints except health checks require a Bearer token:

.. code-block:: bash

    curl -H "Authorization: Bearer YOUR_TOKEN" \
      http://localhost:8080/v1/targets

Generate a strong token:

.. code-block:: bash

    openssl rand -hex 32

Best Practices
--------------

**Use environment variables**
    Never hardcode tokens in config.yaml. Use the ``SERVER_TOKEN`` environment variable instead.

**Read-only volumes**
    Mount config.yaml and OCI keys as read-only:

    .. code-block:: bash

        -v /path/to/config.yaml:/etc/oci-sd/config.yaml:ro
        -v /path/to/keys:/etc/oci-sd/keys:ro

**Restrict key permissions**
    Keep OCI API keys with strict permissions:

    .. code-block:: bash

        chmod 600 oci-keys/api_key.pem

**Unencrypted keys**
    For automation, use unencrypted API keys. This simplifies deployment without compromising security (keys are read-only and not cached in memory beyond startup).

**Secrets management**
    In production, use a secrets manager:
    - AWS Secrets Manager
    - HashiCorp Vault
    - Kubernetes Secrets
    - GitHub Secrets

**Network isolation**
    Run the service in a private network. Only Prometheus servers should access the API.

**Monitor access**
    Enable logging and monitor API access patterns for suspicious activity.

Implementation Details
----------------------

**Timing-safe token comparison**
    Uses ``crypto/subtle.ConstantTimeCompare`` - not vulnerable to timing attacks.

**Minimal runtime image**
    Uses distroless base image (``gcr.io/distroless/static-debian12:nonroot``):
    - No shell
    - No package manager
    - Minimal attack surface

**Key handling**
    - Keys read at startup only
    - Not cached in memory beyond initial read
    - No sensitive data logged
