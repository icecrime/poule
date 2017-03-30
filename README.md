Poule
=====

[![Go Report Card](https://goreportcard.com/badge/github.com/icecrime/poule)](https://goreportcard.com/report/github.com/icecrime/poule) [![CircleCI](https://circleci.com/gh/icecrime/poule.svg?style=svg)](https://circleci.com/gh/icecrime/poule)

# Description

:chicken: Poule is a tool that helps automating tasks on GitHub issues and pull requests. It allows
implementing snippets of behavior (called [**operations**](#operations)) once and be able to invoke them in three
different contexts:

  1. As a one-time operation, on the entire stock of GitHub items.
  2. As part of a batch job alongside multiple other operations.
  3. As part of a long-running daemon, either triggered by GitHub webhooks or scheduled.

The project was created to manage automation on the [Docker project](https://github.com/docker/docker/blob/master/poule.yml).

# Usage

## 1. One-time operations

The following command runs the `dco-check` operation every pull requests in the `docker/docker`
repository without applying any modifications (`dry-run=true`):

```bash
$> poule --repository docker/docker --token-file ~/.gitthub-token --dry-run=true dco-check --filter is:pr
```

## 2. Batch job

The following configuration file describes a `dco-check` operation to run on the `docker/docker`
repository, and can be executed with the `poule batch` command:

```bash
$> cat dco-check.yml
repository: docker/docker

operations:
    - type: dco-check
      filters: {
          is: "pr"
      }

$> poule --token-file ~/.github-token --dry-run=true batch dco-check.yml
```

## 3. Long running job

Poule can either listen on HTTP for GitHub webhooks or use [NSQ](https://nsq.io) as the source of
events. In this mode, actions will be performed based on the configuration as they are received.

See
[`config/serve.example.yml`](https://github.com/icecrime/poule/blob/master/config/serve.example.yml)
for an example configuration. In this mode, each repository can optionally configure its own set of
rules by adding a `poule.yml` file at the root of the source tree.

# Operations

Operations are snippets of GitHub automation.

- An operation is idempotent.
- An operation can apply to GitHub issues, pull requests, or both.
- An operation must implement the [`Operation`
  interface](https://github.com/icecrime/poule/blob/master/src/poule/operations/operations.go).
- A catalog of builtin operations is provided and documented in the [`catalog`
  package](https://github.com/icecrime/poule/tree/master/src/poule/operations/catalog).
