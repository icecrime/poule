Poule operations
================

| Operation        | Issues                  | Pull Requests           | Purpose                                                             |
|------------------|:-----------------------:|:-----------------------:|---------------------------------------------------------------------|
| `ci-label-audit` |                         | :ballot_box_with_check: | Audit CI failure labels and report inconsistencies.                 |
| `ci-label-clean` |                         | :ballot_box_with_check: | Remove CI failures labels where necessary.                          |
| `dco-check`      |                         | :ballot_box_with_check: | Check for commit signatures, label and post a comment if missing.   |
| `label`          | :ballot_box_with_check: | :ballot_box_with_check: | Auto-label issues and pull requests according on matching regexps.  |
| `version-label`  | :ballot_box_with_check: |                         | Add a `version/x` label based on Docker version string in the body. |

## ci-label-audit

## ci-label-clean

## dco-check

## label

Apply a label when the body of the GitHub issue or pull requests matches any of a list of provided
regular expressions.

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

## prune

## rebuild

Rebuild triggers a rebuild operation on pull requests, optionally restricting to a given set of
configurations or statuses (e.g., "documentation" configuration in status "failing"). The `rebuild`
operation also takes an optional `label` parameter which is used as a signal: the operation looks
for the label, triggers a rebuild, and removes it.

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

## version-label
