Building & Development
======================

Prerequisites
-------------

- Go 1.22 or later
- Make
- Docker (for building container images)

Building Binary
---------------

.. code-block:: bash

    git clone https://github.com/amaanx86/oci-prometheus-sd-proxy.git
    cd oci-prometheus-sd-proxy

    # Download dependencies
    make tidy

    # Build binary
    make build

    # Binary at ./bin/oci-sd-proxy
    ./bin/oci-sd-proxy

Running Locally
---------------

.. code-block:: bash

    # Set up configuration
    cp config.yaml.example config.yaml
    # Edit config.yaml with your OCI credentials

    # Set token and run
    SERVER_TOKEN=$(openssl rand -hex 32) make run

    # Test
    curl -H "Authorization: Bearer $SERVER_TOKEN" \
      http://localhost:8080/v1/targets

Testing
-------

Run tests with race detector:

.. code-block:: bash

    make test

Linting
-------

Check code quality:

.. code-block:: bash

    make lint

Docker Image
------------

Build Docker image locally:

.. code-block:: bash

    make docker

Multi-architecture build:

.. code-block:: bash

    docker buildx build --platform linux/amd64,linux/arm64 -t myimage:latest .

Make Targets
------------

.. code-block:: bash

    make build           # Build binary
    make test            # Run tests
    make lint            # Lint code
    make docker          # Build Docker image
    make tidy            # Tidy dependencies
    make run             # Run locally (requires SERVER_TOKEN env var)
    make clean           # Clean build artifacts
