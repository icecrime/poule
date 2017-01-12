Poule operations
================

| Operation           | Docker specific | Issues                  | Pull Requests           | Purpose                                                             |
|---------------------|:---------------:|:-----------------------:|:-----------------------:|---------------------------------------------------------------------|
| `ci-label-clean`    |                 |                         | :ballot_box_with_check: | Remove CI failures labels where necessary.                          |
| `dco-check`         | :whale:         |                         | :ballot_box_with_check: | Check for commit signatures, label and post a comment if missing.   |
| `label`             |                 | :ballot_box_with_check: | :ballot_box_with_check: | Auto-label issues and pull requests according on matching regexps.  |
| `poule-updater`     |                 |                         | :ballot_box_with_check: | Reload `poule` configuration when a pull request modifies it.       |
| `prune  `           |                 | :ballot_box_with_check: |                         | Manage issues with no activities.                                   |
| `random-assign`     |                 | :ballot_box_with_check: | :ballot_box_with_check: | Auto-assign a random user to issues and pull requests.              |
| `rebuild`           | :whale:         |                         | :ballot_box_with_check: | Rebuild all or selected pull request jobs.                          |
| `version-label`     | :whale:         | :ballot_box_with_check: |                         | Add a `version/x` label based on Docker version string in the body. |
| `version-milestone` | ~               |                         | :ballot_box_with_check: | Add merged pull requests to the upcoming milestone.                 |

## CI label cleaning

## DCO check

## Label

The `label` operation applies a label when the body of the GitHub issue or pull requests matches any
of a list of provided regular expressions.

#### Configuration

| Configuration     | Description                                                                                                        |
|-------------------|--------------------------------------------------------------------------------------------------------------------|
| `patterns`        | A map of string to string arrays, where keys are the label to add, and values are a collection of regexp to match. |

#### Example configuration

```yaml
type: label
filters: {
    is: "issue",
}
settings: {
    patterns: {
        platform/desktop:    [ "docker for mac", "docker for windows" ],
        platform/freebsd:    [ "freebsd" ],
        platform/windows:    [ "nanoserver", "windowsservercore", "windows server" ],
    }
}
```

## Poule update

The `poule-updater` operation is a very special one that monitors for merged pull request which
modify a `poule.yml` file at the root of the repository, and reload the internal configuration
accordingly. It is only meant to be used as a server-mode operation.

## Prune

## Random assign

#### Configuration

| Configuration  | Description      |
|----------------|------------------|
| `users`        | A string array.  |

#### Example configuration

```yaml
type: random-assign
settings: {
    users: ["icecrime", "vieux"]
}
```

## Rebuild

The `rebuild` operation triggers a rebuild operation on pull requests, optionally restricting to a
given set of configurations or statuses (e.g., "documentation" configuration in status "failing").
The `rebuild` operation also takes an optional `label` parameter which is used as a signal: the
operation looks for the label, triggers a rebuild, and removes it.

Example use cases:
- As a one-time command invokation to rebuild pull requests after a test was fixed.
- In server mode, to trigger a rebuild when a given label is set on a pull request.

#### Configuration

| Configuration     | Description                                                                                                        |
|-------------------|--------------------------------------------------------------------------------------------------------------------|
| `configurations`  | The list of configurations to consider for rebuild (empty means all).                                              |
| `label`           | Filter pull requests with the corresponding label, and remove it after the rebuild is triggered.                   |
| `statuses`        | The list of status to meet for rebuilding (default: `[ "failing", "error" ]`).                                     |

#### Example configuration

```yaml
type: rebuild
settings: {
    configurations: [ janky ]
    label:          "rebuild/janky"
    statuses:       [ error, failing ]
}
```

## Version label

## Version milestone

The `version-milestone` operation adds merged pull requests to the currently active milestone: it uses
a `VERSION` file at the root of the repository to determine the current version in the making and
searches for a milestone which title matches that version string.

The goal is for every merged pull request to be attached to a milestone, in such way that it's
trivial to determine from GitHub in which release a given changeset was shipped.

