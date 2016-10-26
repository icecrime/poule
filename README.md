Poule
=====

# /!\ Work in progress /!\

# Description

:chicken: Poule is a tool that helps automating tasks on GitHub issues and pull
requests. The intent is to implement the behavior to automate once, and to be
able to run it in three different contexts:

  1. As a one-time operation, on the entire stock of GitHub items.
  2. As part of a batch job alongside multiple other operations.
  3. As part of a long-running job, triggered by GitHub webhooks.

## 1. One-time operations

Example:

```bash
$> poule --repository docker/docker --token-file ~/.gitthub-token --dry-run=true dco-check --filter is:pr
```

## 2. Batch job

Example:

```bash
$> cat dco-check.toml
dry_run = true
repository = "docker/docker"
token = "<github-token>"

[[operations]]
type = "dco-check"
[operations.filters]
is = "pr"

$> poule batch dco-check.yml
```

## 3. Long running job

Poule can connect to NSQ to receive events for Github issues and pull requests.
It will then use this event data to perform the actions listed in a config
file.

## 4. Poule Server
Example Config:

```toml
dry_run = true
listen_addr = ":8080"
nsq_lookupd_addr = "127.0.0.1:4150"
token = "<github-api-token>"
topic = "pulls"
channel = "poule"

[triggers.default]
repositories = ["docker/docker"]

[[triggers.default.events]]
type = "pull_request"
action = "opened"

[[triggers.default.events]]
type = "pull_request"
action = "reopened"

[[triggers.default.operations]]
type = "dco-check"
[triggers.default.operations.filters]
is = "pr"

```

```bash
$> poule serve config.toml
```
