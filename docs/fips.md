# FIPS image and government endpoint configuration

The operator image builds with `GOFIPS140=v1.0.0` and runs with
`GODEBUG=fips140=on` by default. The container release workflow publishes the
usual `coralogixrepo/coralogix-operator:vX.Y.Z` tags for Linux amd64 and arm64.
Helm uses those same tags without an additional FIPS value or tag suffix.

The image retains the Go 1.26 builder, `CGO_ENABLED=0`, and distroless static
runtime base. It uses Go's native cryptographic module. Makefile builds and
unit tests also default to the pinned module, and the CI test image uses the
same runtime setting as the release image.

FIPS mode restricts TLS negotiation to approved protocol versions, cipher suites,
signature algorithms, and key exchanges. Test connectivity to the Coralogix and
Kubernetes API endpoints with the default image in the target environment.

## Deploy with Helm

Use an operator release that includes this default. Previously published images
are unchanged. The image version defaults to the chart's `appVersion`, or can
be overridden with `coralogixOperator.image.tag` (without the leading `v`).
The values below are also supported by the local chart at
`./charts/coralogix-operator` when testing an unreleased build.

```yaml
coralogixOperator:
  # image:
  #   tag: "X.Y.Z" # Optional published operator version, without the leading v.
  region: ""
  domain: "gov.example.com" # Replace with your assigned Coralogix domain.
secret:
  create: false
  secretKeyReference:
    name: coralogix-operator-api-key
    key: apiKey
```

```sh
helm upgrade --install coralogix-operator coralogix/coralogix-operator \
  --namespace coralogix-operator-system --create-namespace \
  --values fips-values.yaml
```

Create the referenced API key Secret in that namespace before installing.
The domain in this example is a reserved placeholder, not a Coralogix endpoint.
Obtain the correct government domain and API key from your account administrator.

There is no named government region in the operator's region list. The existing
domain setting supports custom domains, so no region code change is needed.
Set exactly one of `region` or `domain`; when upgrading a region-based deployment,
explicitly clear `region`. Supply a hostname without an `https://` scheme or path.
The SDK adds `api.` if absent and uses `https://api.<domain>/mgmt/openapi/5`;
an already `api.`-prefixed hostname is also accepted. Direct deployments can use
`--domain` or `CORALOGIX_DOMAIN` and must clear any region flag/environment value.

## Validation status and compliance review

Verified on October 6, 2026: Go Cryptographic Module v1.0.0 is covered by
[NIST CMVP certificate #5247](https://csrc.nist.gov/projects/cryptographic-module-validation-program/certificate/5247),
which is active for FIPS 140-3, overall level 1, with a sunset date of April 26,
2031. [Go's FIPS documentation](https://go.dev/doc/security/fips140) identifies
that certificate for v1.0.0 and explains the build and runtime settings.

This image configuration enables use of the validated module. It does not by
itself establish FedRAMP compliance or authorization for the operator or a
customer deployment. Before making a customer compliance claim, the compliance
owner and auditors must confirm that the module, its approved services, and the
actual deployment satisfy the applicable requirements. Review the
[module security policy](https://csrc.nist.gov/CSRC/media/projects/cryptographic-module-validation-program/documents/security-policies/140sp5247.pdf),
including the module boundary, operating environment, source integrity,
cryptographic services, and key/entropy requirements. Include the Kubernetes
host OS and hardware, container environment, TLS endpoints and certificates,
and any cryptography outside the Go module in that assessment.

Keep release image digests, compiler/build metadata, runtime configuration, and
deployment test evidence for the assessment. Do not override the image's
`GODEBUG` with `fips140=off`. If adding other `GODEBUG` options, retain
`fips140=on`. The testing-only `fips140=only` mode is not intended for production.

## Local build and inspection

Use Docker Buildx/BuildKit to build the default image:

```sh
docker buildx build --load --platform linux/amd64 \
  --tag coralogix-operator-local:fips .
docker image inspect coralogix-operator-local:fips --format '{{json .Config.Env}}'
docker run --rm coralogix-operator-local:fips --help
```

Use `linux/arm64` on an arm64 host. The environment must include
`GODEBUG=fips140=on`, and `--help` should exit successfully after initialization.
To inspect the binary, copy `/manager` from a stopped container and run
`go version -m` on it:

```sh
container_id=$(docker create coralogix-operator-local:fips)
docker cp "$container_id:/manager" ./manager-fips
docker rm "$container_id"
go version -m ./manager-fips
rm ./manager-fips
```

The output should include `GOFIPS140=v1.0.0-c2097c7c` (the toolchain's normalized
name for the requested module) and `CGO_ENABLED=0`. Compare the frozen module
archive's SHA-256 with section 11.1 of the security policy when collecting
release evidence. A successful build and startup are engineering checks;
endpoint reconciliation and auditor approval still require deployment-specific
validation.
