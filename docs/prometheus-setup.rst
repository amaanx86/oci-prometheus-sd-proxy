Prometheus Setup
================

Configuration
-------------

Add HTTP SD configuration to your ``prometheus.yml``:

.. code-block:: yaml

    scrape_configs:
      - job_name: oci_instances
        http_sd_configs:
          - url: 'http://oci-sd-proxy:8080/v1/targets'
            authorization:
              type: Bearer
              credentials: 'YOUR_SERVER_TOKEN'

Relabeling
----------

Map OCI metadata to Prometheus labels:

.. code-block:: yaml

    relabel_configs:
      # Instance name
      - source_labels: [__meta_oci_instance_name]
        target_label: instance

      # Tenancy
      - source_labels: [__meta_oci_tenancy_name]
        target_label: tenancy

      # Region
      - source_labels: [__meta_oci_region]
        target_label: region

      # Compartment
      - source_labels: [__meta_oci_compartment_id]
        target_label: compartment

      # Shape
      - source_labels: [__meta_oci_shape]
        target_label: shape

      # Availability domain
      - source_labels: [__meta_oci_availability_domain]
        target_label: availability_domain

Available Labels
----------------

All discovered instances include these labels:

- ``__meta_oci_instance_name`` - Instance name
- ``__meta_oci_instance_id`` - Instance OCID
- ``__meta_oci_instance_state`` - RUNNING, STOPPED, etc.
- ``__meta_oci_tenancy_name`` - Tenancy name
- ``__meta_oci_tenancy_id`` - Tenancy OCID
- ``__meta_oci_region`` - OCI region
- ``__meta_oci_compartment_id`` - Compartment OCID
- ``__meta_oci_shape`` - Instance shape (e.g., VM.Standard.E6.Flex)
- ``__meta_oci_availability_domain`` - AD name
- ``__meta_oci_fault_domain`` - Fault domain
- ``__meta_oci_image_id`` - Image OCID
- ``__meta_oci_private_ip`` - Private IP address
- ``__meta_oci_tag_*`` - All custom OCI tags (e.g., ``__meta_oci_tag_env``)

Filtering
---------

Use metric_relabel_configs to filter targets:

.. code-block:: yaml

    metric_relabel_configs:
      # Skip stopped instances
      - source_labels: [__meta_oci_instance_state]
        regex: '^STOPPED$'
        action: drop

      # Only keep Linux instances
      - source_labels: [__meta_oci_tag_os]
        regex: '^linux$'
        action: keep

Testing
-------

Test the service discovery endpoint:

.. code-block:: bash

    TOKEN=your_server_token
    curl -H "Authorization: Bearer $TOKEN" \
      http://localhost:8080/v1/targets | jq .

Example output:

.. code-block:: json

    [
      {
        "targets": ["10.0.1.5:9100"],
        "labels": {
          "__meta_oci_instance_name": "prod-web-01",
          "__meta_oci_tenancy_name": "my-tenancy",
          "__meta_oci_region": "me-jeddah-1",
          "__meta_oci_shape": "VM.Standard.E6.Flex"
        }
      }
    ]
