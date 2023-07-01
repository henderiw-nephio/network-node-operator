# network-node

## Description
network-node controller

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] network-node`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree network-node`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init network-node
kpt live apply network-node --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
