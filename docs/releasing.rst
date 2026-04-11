Release Process
===============

This document covers how to cut a release, how image signing works, and how a new maintainer can be onboarded as a TUF signer.

Overview
--------

Every release goes through the following steps in order:

1. Merge all changes to ``main`` via pull request
2. Update ``CHANGELOG.md`` - move ``[Unreleased]`` entries into a versioned section
3. Create a GitHub release using the changelog content as release notes (triggers the build workflow)
4. The CI workflow builds, signs, and attests the image automatically
5. Sign the TUF targets metadata locally **before** merging the TUF signing PR
6. Trigger the TUF publish workflow to deploy updated metadata to GitHub Pages


Step-by-step
------------

1. Prepare the changelog
~~~~~~~~~~~~~~~~~~~~~~~~

In ``CHANGELOG.md``, rename ``## [Unreleased]`` to the new version and date, then add a fresh empty ``## [Unreleased]`` section above it::

    ## [Unreleased]

    ## [1.5.0] - 2026-06-01

    ### Added
    - ...

Commit on a feature branch and merge to ``main`` via PR.

2. Create the GitHub release
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Use the changelog section as the release notes body. This creates the tag and triggers the ``Build and Push Docker Image`` workflow::

    gh release create v1.5.0 \
      --title "v1.5.0" \
      --notes "## Added
    - Feature one
    - Feature two

    ## Changed
    - Change one

    **Full Changelog**: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.4.1...v1.5.0" \
      --target main

Paste the relevant ``CHANGELOG.md`` section directly into ``--notes``. Include the ``Full Changelog`` comparison link at the bottom using the previous tag and the new tag.

The workflow will:

- Build a multi-arch image (``linux/amd64``, ``linux/arm64``) and push to GHCR
- Generate a CycloneDX SBOM via ``anchore/sbom-action`` and attach it as a cosign attestation (``--type cyclonedx``); also uploaded as a release asset
- Sign the image digest with cosign using GitHub Actions OIDC (keyless)
- Generate ``release-metadata.json`` (digest, commit, tag, build timestamp) and attach it as a cosign attestation under the project-specific predicate type
- Upload ``release-metadata.json`` as a release asset
- Push a ``sign/release-1-5-0-<run_id>`` branch to the TUF repository
- Generate SLSA L3 build provenance via ``slsa-github-generator`` in a separate isolated job with its own OIDC identity

3. Sign the TUF targets metadata
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. important::

   Complete this step **before** merging the TUF signing PR.
   Merging without signing leaves ``metadata/targets.json`` out of date.

The CI push triggers the TUF signing event workflow, which opens a pull request in ``amaanx86/oci-prometheus-sd-proxy-tuf-on-ci``. Check the PR to get the exact branch name, then from the TUF repository::

    cd ~/github/oci-prometheus-sd-proxy-tuf-on-ci
    source .venv-tuf/bin/activate

    tuf-on-ci-sign sign/release-1-5-0-<run_id>

A browser window opens for Sigstore authentication. Sign in as ``@amaanx86``. The tool updates ``metadata/targets.json`` to include the new release manifest with its SHA-256 hash, signs the metadata, and pushes the signature back to the branch.

Confirm the target is listed::

    cat metadata/targets.json | python3 -m json.tool | grep -A5 "v1.5.0"

4. Merge the TUF signing PR
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Once the ``TUF-on-CI signing event`` check passes, merge the PR. The ``online-sign`` workflow runs automatically and refreshes ``snapshot`` and ``timestamp``.

5. Publish TUF metadata to GitHub Pages
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

    gh workflow run publish.yml \
      --repo amaanx86/oci-prometheus-sd-proxy-tuf-on-ci


Verifying a release
--------------------

Every release produces four independently verifiable artifacts: an image signature, a CycloneDX SBOM attestation, a release-metadata attestation, and SLSA L3 build provenance. A TUF metadata chain separately records the authorised release digest and can be inspected directly at https://github.com/amaanx86/oci-prometheus-sd-proxy-tuf-on-ci. All verification uses the image digest rather than the tag; the tag is a mutable pointer but the digest is what signatures actually cover.

Tools required: ``cosign`` (`install <https://docs.sigstore.dev/cosign/system_config/installation/>`_), ``slsa-verifier`` (`install <https://github.com/slsa-framework/slsa-verifier#installation>`_), and ``python3``.

Step 0: resolve the digest
~~~~~~~~~~~~~~~~~~~~~~~~~~

All commands below require the image digest. Resolve it once and reuse it::

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

Step 1: image signature
~~~~~~~~~~~~~~~~~~~~~~~~

Verifies that the image digest was signed by the ``docker-build-push.yml`` workflow for this exact tag, with the event recorded in the Rekor transparency log::

    cosign verify "${IMAGE}" \
      --certificate-identity \
        "https://github.com/amaanx86/oci-prometheus-sd-proxy/.github/workflows/docker-build-push.yml@refs/tags/v<version>" \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com

Step 2: SBOM attestation
~~~~~~~~~~~~~~~~~~~~~~~~~

Verifies the CycloneDX SBOM attestation. The SBOM payload is printed to stdout on success (pipe to ``| python3 -m json.tool`` to read it)::

    cosign verify-attestation "${IMAGE}" \
      --certificate-identity \
        "https://github.com/amaanx86/oci-prometheus-sd-proxy/.github/workflows/docker-build-push.yml@refs/tags/v<version>" \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com \
      --type cyclonedx \
      > /dev/null

Step 3: release-metadata attestation
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Verifies the in-toto attestation containing the image digest, source commit, release tag, and build timestamp::

    cosign verify-attestation "${IMAGE}" \
      --certificate-identity \
        "https://github.com/amaanx86/oci-prometheus-sd-proxy/.github/workflows/docker-build-push.yml@refs/tags/v<version>" \
      --certificate-oidc-issuer https://token.actions.githubusercontent.com \
      --type "https://github.com/amaanx86/oci-prometheus-sd-proxy/release-metadata" \
      | python3 -c "
    import json, sys, base64
    e = json.load(sys.stdin)
    p = e['payload']; p += '=' * (-len(p) % 4)
    stmt = json.loads(base64.b64decode(p))
    print(json.dumps(stmt['predicate'], indent=2))
    "

Step 4: SLSA L3 provenance
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Verifies that the image was built by the ``slsa-github-generator`` trusted builder in an isolated job from the exact source commit. ``slsa-verifier`` requires a digest reference - it rejects mutable tags by design::

    slsa-verifier verify-image "${IMAGE}" \
      --source-uri "github.com/amaanx86/oci-prometheus-sd-proxy" \
      --source-tag "v<version>"

A passing result prints the verified builder identity and the source commit::

    Verified build using builder "https://github.com/slsa-framework/slsa-github-generator/
      .github/workflows/generator_container_slsa3.yml@refs/tags/v2.0.0"
    at commit <sha>
    PASSED: SLSA verification passed

TUF signing keys
-----------------

This project uses `TUF-on-CI <https://github.com/theupdateframework/tuf-on-ci>`_ for release metadata signing. All keys are keyless Sigstore OIDC keys - no private key files exist anywhere.

.. list-table::
   :header-rows: 1
   :widths: 20 20 60

   * - TUF role
     - Key type
     - Identity
   * - root
     - sigstore-oidc / Fulcio
     - ``@amaanx86`` via GitHub OAuth
   * - targets
     - sigstore-oidc / Fulcio
     - ``@amaanx86`` via GitHub OAuth
   * - snapshot
     - sigstore-oidc / Fulcio
     - GitHub Actions OIDC (``online-sign.yml`` workflow)
   * - timestamp
     - sigstore-oidc / Fulcio
     - GitHub Actions OIDC (``online-sign.yml`` workflow)

Root and targets are **offline** roles - signing requires a browser session by the maintainer. Snapshot and timestamp are **online** roles signed automatically by CI.

Current signers: `@amaanx86 <https://github.com/amaanx86>`_


Adding a new maintainer as a signer
-------------------------------------

When a new maintainer needs to sign releases, their Sigstore identity must be added to the TUF root and targets roles. This requires ``@amaanx86`` to co-sign the root rotation.

Prerequisites for the new maintainer:

- A GitHub account
- The TUF repository cloned locally
- ``tuf-on-ci`` installed (``pip3 install tuf-on-ci``)
- Write access to ``amaanx86/oci-prometheus-sd-proxy-tuf-on-ci``

Steps:

1. From the TUF repository, create a delegation signing event::

       cd ~/github/oci-prometheus-sd-proxy-tuf-on-ci
       source .venv-tuf/bin/activate
       tuf-on-ci-delegate targets --signer <github-username>

   This creates a ``sign/add-signer-<username>`` branch.

2. Both ``@amaanx86`` and the new maintainer run ``tuf-on-ci-sign`` on this branch - ``@amaanx86`` to authorize the delegation, the new maintainer to record their identity::

       tuf-on-ci-sign sign/add-signer-<username>

3. Merge the signing PR after both signatures are present.

4. The new maintainer can now sign future release branches using their own GitHub identity.

No key material is ever shared. Each signer authenticates independently.


TUF repository
--------------

- Signing repo: https://github.com/amaanx86/oci-prometheus-sd-proxy-tuf-on-ci
- Published metadata (GitHub Pages): https://amaanx86.github.io/oci-prometheus-sd-proxy-tuf-on-ci/
