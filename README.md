# pb

Fast CLI for Productboard, built in Go against the v2 API.

## Install

```bash
go install .
```

The binary is installed as `pb-cli`. Symlink or copy it to `pb`:

```bash
ln -sf $(go env GOPATH)/bin/pb-cli $(go env GOPATH)/bin/pb
```

## Auth

Set your API token via env var:

```bash
export PRODUCTBOARD_API_TOKEN="your-token"
```

Or write it to `~/.config/pb/token`:

```bash
mkdir -p ~/.config/pb
echo "your-token" > ~/.config/pb/token
chmod 600 ~/.config/pb/token
```

Or pass it per-command (the longest way):

```bash
pb --token "your-token" features list
```

## Commands

### Features

```bash
pb features list                          # list features (defaults to Agent Assure component)
pb features list --component <id>         # filter by component
pb features list --product <id>           # filter by product
pb features list --status in_progress     # filter by status
pb features list --json                   # raw JSON output
pb features list --id-only               # one ID per line

pb features get <id>                      # get feature detail (JSON)
pb features create --name "My Feature"   # create in Agent Assure
pb features create --name "My Feature" --component <id> --description "<p>HTML</p>"
pb features update <id> --name "New Name"
pb features update <id> --status done
pb features update <id> --health onTrack
pb features update <id> --health atRisk --health-comment "<p>Blocked on infra work</p>"
pb features update <id> --health-comment "<p>Still on track, minor delays</p>"
pb features delete <id>
```

### Products

```bash
pb products list                          # list all products
pb products hierarchy                     # show product/component tree
```

### Releases

```bash
pb releases list
pb releases create --name "v1.0" --description "Release notes"
```

### Notes

```bash
pb notes list
```

### Objectives

```bash
pb objectives list
```

## Global Flags


| Flag        | Description                             |
| ----------- | --------------------------------------- |
| `--json`    | Output raw JSON (for `jq`, scripts)     |
| `--id-only` | One ID per line (for `xargs` pipelines) |
| `--token`   | API token (overrides env/config)        |


## Defaults

All feature operations default to the **Agent Assure** component (`060e336f-2149-4c06-8e42-0c87b7d987b8`) under the **Assurance** product (`c48c1312-9da5-4b75-b6b1-824ce6837894`).

## Development

```bash
go test ./...       # run tests
go build ./...      # build
go generate ./...   # regenerate API clients from specs/
```

API clients are generated from `specs/entities.yaml` and `specs/notes.yaml` using [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). Never edit files under `api/` by hand.