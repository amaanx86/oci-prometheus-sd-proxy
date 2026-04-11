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

Supply Chain Security
---------------------

Every release image is signed and attested through a fully keyless pipeline - no
long-lived private keys exist anywhere. The pipeline produces four independently
verifiable artifacts. Full verification commands are documented in
:doc:`releasing`.

**What is signed**

- Image signature via cosign (GitHub Actions OIDC, recorded in Rekor)
- CycloneDX SBOM attached as a cosign attestation (``--type cyclonedx``)
- Release-metadata attestation (digest, source commit, build timestamp)
- SLSA L3 build provenance via ``slsa-github-generator`` in an isolated job
- TUF metadata chain published to GitHub Pages via ``tuf-on-ci``

**Verifying an image before deployment**

All verification uses the image digest, not the tag. Resolve it first:

.. code-block:: bash

    DIGEST=$(cosign verify \
      --certificate-identity \
        "https://github.com/amaanx86/oci-prometheus-sd-proxy/.github/workflows/docker-build-push.yml@refs/tags/v<version>" \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com \
      ghcr.io/amaanx86/oci-prometheus-sd-proxy:<version> 2>/dev/null \
      | python3 -c "
    import json, sys
    records = json.load(sys.stdin)
    print(list({r['critical']['image']['docker-manifest-digest'] for r in records})[0])
    ")

    IMAGE="ghcr.io/amaanx86/oci-prometheus-sd-proxy@${DIGEST}"

Image signature:

.. code-block:: bash

    cosign verify "${IMAGE}" \
      --certificate-identity \
        "https://github.com/amaanx86/oci-prometheus-sd-proxy/.github/workflows/docker-build-push.yml@refs/tags/v<version>" \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com

SLSA L3 provenance (requires ``slsa-verifier``):

.. code-block:: bash

    slsa-verifier verify-image "${IMAGE}" \
      --source-uri "github.com/amaanx86/oci-prometheus-sd-proxy" \
      --source-tag "v<version>"

See :doc:`releasing` for SBOM attestation and release-metadata attestation verification.

**TUF metadata repository**

Release manifests are published independently at
https://github.com/amaanx86/oci-prometheus-sd-proxy-tuf-on-ci.
This keeps TUF lifecycle management (key rotation, metadata expiry) isolated
from the application repository. The signed metadata is served via GitHub Pages at
https://amaanx86.github.io/oci-prometheus-sd-proxy-tuf-on-ci/metadata/.

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
