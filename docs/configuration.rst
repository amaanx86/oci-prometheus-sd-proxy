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
     - monitoring
     - OCI tag key to filter instances
   * - ``DISCOVERY_TAG_VALUE``
     - enabled
     - OCI tag value to filter instances
   * - ``DISCOVERY_LINUX_PORT``
     - 9100
     - Port for Linux node_exporter
   * - ``DISCOVERY_WINDOWS_PORT``
     - 9182
     - Port for Windows exporter
   * - ``DISCOVERY_REFRESH_INTERVAL``
     - 5m
     - How often to poll OCI APIs

config.yaml
-----------

Main configuration file with OCI tenancy credentials:

.. code-block:: yaml

    server:
      port: 8080
      token: "use-SERVER_TOKEN-env-var"

    discovery:
      tag_key: monitoring
      tag_value: enabled
      linux_port: 9100
      windows_port: 9182
      refresh_interval: 5m

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
    OCI freeform tag for filtering instances (e.g., ``monitoring=enabled``)

**discovery.linux_port**
    Port for Linux Prometheus exporters (default: node_exporter on 9100)

**discovery.windows_port**
    Port for Windows Prometheus exporters (default: windows_exporter on 9182)

**discovery.refresh_interval**
    Background cache refresh interval (e.g., ``5m``, ``30s``)

**tenancies[]**
    List of OCI tenancies to discover from

**tenancies[].name**
    Friendly name (used in ``__meta_oci_tenancy_name`` label)

**tenancies[].region**
    OCI region code (e.g., ``me-jeddah-1``, ``us-ashburn-1``)

**tenancies[].tenancy_id**
    Tenancy OCID

**tenancies[].user_id**
    User OCID for API authentication

**tenancies[].fingerprint**
    API key fingerprint

**tenancies[].private_key_path**
    Path to unencrypted PEM private key

**tenancies[].passphrase**
    Passphrase for encrypted keys (leave empty for unencrypted)

**tenancies[].compartments**
    List of compartment OCIDs to scan. Leave empty ``[]`` to auto-discover all compartments.
