# Registry index

This directory is the **static registry index** of the Flutter delivery
lifecycle standard (ADR-030): one registry metadata document per released
version, laid out as `registry/index/anvil-standard-flutter/<version>.json`
— the index layout of the Anvil Runtime registry client (Core
`internal/registry/index.go`).

The Release workflow ([`.github/workflows/release.yml`](../.github/workflows/release.yml))
adds exactly one document per published release (add-only; small diffs;
parallel releases never rewrite shared files). The directory must contain
only index documents: every `*.json` file under it is treated as an entry
document by the registry client — do not place other JSON files here.

Discover releases from a checkout of this repository:

```sh
anvil standard list --index <checkout-of-this-repository>
anvil standard inspect anvil-standard-flutter <version> --index <checkout-of-this-repository>
```
