Configuration
==============

Environment Variables
---------------------

All configuration can be set via environment variables:

.. list-table::
   :header-rows: 1

   * - Variable
     - Default
     - Description
   * - ``SERVER_PORT``
     - 8080
     - HTTP listen port
   * - ``SERVER_TOKEN``
     - (required)
     - Bearer token for API authentication
   * - ``CONFIG_PATH``
     - config.yaml
     - Path to configuration file
   * - ``DISCOVERY_TAG_KEY``
     - (empty)
     - OCI tag key to filter instances. When both ``tag_key`` and ``tag_value`` are empty, all running instances are returned without tag filtering.
   * - ``DISCOVERY_TAG_VALUE``
     - (empty)
     - OCI tag value to filter instances. When both ``tag_key`` and ``tag_value`` are empty, all running instances are returned without tag filtering.
   * - ``DISCOVERY_LINUX_PORT``
     - 9100
     - Port for Linux node_exporter
   * - ``DISCOVERY_WINDOWS_PORT``
     - 9182
     - Port for Windows exporter
   * - ``DISCOVERY_REFRESH_INTERVAL``
     - 5m
     - How often to poll OCI APIs
   * - ``DISCOVERY_RATE_LIMIT_RPS``
     - 10.0
     - OCI API rate limit (requests/sec per tenancy)

config.yaml
-----------

Main configuration file with OCI tenancy credentials:

.. code-block:: yaml

    server:
      port: 8080
      token: "use-SERVER_TOKEN-env-var"

    discovery:
      # tag_key: monitoring
      # tag_value: enabled
      # Omit both tag_key and tag_value to discover all running instances
      linux_port: 9100
      windows_port: 9182
      refresh_interval: 5m
      rate_limit_rps: 10.0

    tenancies:
      - name: my-tenancy
        region: me-jeddah-1
        tenancy_id: ocid1.tenancy.oc1..xxxxxx
        user_id: ocid1.user.oc1..xxxxxx
        fingerprint: "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
        private_key_path: /etc/oci-sd/keys/api_key.pem
        passphrase: ""
        compartments: []  # Empty = auto-discover all

Fields
~~~~~~

**server.port**
    HTTP port to listen on

**server.token**
    Bearer token (prefer ``SERVER_TOKEN`` environment variable)

**discovery.tag_key / tag_value**
    OCI freeform or defined tag for filtering instances (e.g., ``monitoring=enabled``).
    Both fields are optional. When both are omitted (or empty), the proxy returns all
    running instances without tag filtering - useful for discovering everything in a
    region and relying on Prometheus relabel rules instead.

**discovery.linux_port**
    Port for Linux Prometheus exporters (default: node_exporter on 9100)

**discovery.windows_port**
    Port for Windows Prometheus exporters (default: windows_exporter on 9182)

.. note::

   **Windows OS detection** - the proxy selects the port using this priority order:

   1. OCI freeform tag ``os = windows`` on the instance (highest priority)
   2. Instance display name contains ``win`` (e.g. ``win-server-01``, ``windows-web``)
   3. Everything else defaults to ``linux_port`` (9100)

   If a Windows VM has no ``os`` tag and no ``win`` in its display name, it will be
   targeted on port 9100. To avoid this, either set the freeform tag ``os = windows``
   on the OCI instance, or ensure ``win`` appears in the VM display name.

   When installing ``windows_exporter`` via the MSI installer, configure it to listen
   on port 9182 (the default). If you prefer port 9100 for Windows, set that in the
   MSI installer and update ``windows_port`` in this config to match.

**discovery.refresh_interval**
    Background cache refresh interval (e.g., ``5m``, ``30s``)

**discovery.rate_limit_rps**
    OCI API rate limit in requests per second per tenancy. Prevents 429 TooManyRequests errors by proactively throttling requests. Combined with automatic retry policy for transient failures. (default: ``10.0``)

**tenancies[]**
    List of OCI tenancies to discover from

**tenancies[].name**
    Friendly name (used in ``__meta_oci_tenancy_name`` label)

**tenancies[].region**
    OCI region code (e.g., ``me-jeddah-1``, ``us-ashburn-1``)

**tenancies[].tenancy_id**
    Tenancy OCID

**tenancies[].auth_type**
    Authentication method. One of:

    - ``api_key`` (default) - static user credentials; requires ``user_id``, ``fingerprint``, and ``private_key_path``
    - ``instance_principal`` - authenticates via OCI IMDS; no credential fields needed. Only valid when the proxy runs on an OCI compute instance with a dynamic group and IAM policy granting read access. See :doc:`installation` for setup steps.

**tenancies[].user_id**
    User OCID for API authentication. Required when ``auth_type`` is ``api_key``.

**tenancies[].fingerprint**
    API key fingerprint. Required when ``auth_type`` is ``api_key``.

**tenancies[].private_key_path**
    Path to unencrypted PEM private key. Required when ``auth_type`` is ``api_key``.

**tenancies[].passphrase**
    Passphrase for encrypted keys (leave empty for unencrypted). Only used with ``api_key`` auth.

**tenancies[].compartments**
    List of compartment OCIDs to scan. Leave empty ``[]`` to auto-discover all compartments.
