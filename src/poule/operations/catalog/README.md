Poule operations
================

Catalog of operations:

| Operation        | Issues                  | Pull Requests           | Purpose                                                             |
|------------------|:-----------------------:|:-----------------------:|---------------------------------------------------------------------|
| `ci-label-audit` |                         | :ballot_box_with_check: | Audit CI failure labels and report inconsistencies.                 |
| `ci-label-clean` |                         | :ballot_box_with_check: | Remove CI failures labels where necessary.                          |
| `dco-check`      |                         | :ballot_box_with_check: | Check for commit signatures, label and post a comment if missing.   |
| `label`          | :ballot_box_with_check: | :ballot_box_with_check: | Auto-label issues and pull requests according on matching regexps.  |
| `version-label`  | :ballot_box_with_check: |                         | Add a `version/x` label based on Docker version string in the body. |

# ci-label-audit

# ci-label-clean

# dco-check

# label

Apply a label when the body of the GitHub issue or pull requests matches any of a list of provided
regular expressions.

### Configuration

| Configuration     | Description                                                                                                              |
|-------------------|--------------------------------------------------------------------------------------------------------------------|
| `patterns`        | A map of string to string arrays, where keys are the label to add, and values are a collection of regexp to match. |

### Example configuration

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

# prune

# rebuild

### Example configuration

```yaml
type: rebuild
settings: {
    configurations: [ experimental, win2lin ]
}
```

# version-label
