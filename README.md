Poule
=====

[![CircleCI](https://circleci.com/gh/icecrime/poule.svg?style=svg)](https://circleci.com/gh/icecrime/poule)

# Description

:chicken: Poule is a tool that helps automating tasks on GitHub issues and pull requests. The intent
is to implement snippets of behavior, called **operations**, once and be able to invoke it from
three different contexts:

  1. As a one-time operation, on the entire stock of GitHub items.
  2. As part of a batch job alongside multiple other operations.
  3. As part of a long-running daemon, triggered by GitHub webhooks.

### 1. One-time operations

Example:

```bash
$> poule --repository docker/docker --token-file ~/.gitthub-token --dry-run=true dco-check --filter is:pr
```

### 2. Batch job

Example:

```bash
$> cat dco-check.yml
repository: docker/docker

operations:
    - type: dco-check
      filters: {
          is: "pr"
      }
```

### 3. Long running job

Poule can connect to NSQ to receive events for Github issues and pull requests.  It will then use
this event data to perform the actions listed in a config file.

See
[`config/serve.example.yml`](https://github.com/icecrime/poule/blob/master/config/serve.example.yml)
for an example configuration.

# Operations

Operations are snippets of GitHub automation.

- An operation is idempotent.
- An operation can apply to GitHub issues, pull requests, or both.
- An operation must implement the [`Operation`
  interface](https://github.com/icecrime/poule/blob/master/src/poule/operations/operations.go).
- A catalog of builtin operations is provided and documented in the [`catalog`
  package](https://github.com/icecrime/poule/tree/master/src/poule/operations/catalog).
