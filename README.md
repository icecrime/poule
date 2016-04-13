Poule
=====

```
NAME:
   poule - Mass interact with GitHub pull requests

USAGE:
   poule [global options] command [command options] [arguments...]
   
VERSION:
   0.1.0
   
COMMANDS:
    audit       audit github jobs failure
    clean       clean github failure labels
    rebuild     rebuild failed jobs

GLOBAL OPTIONS:
   --repository         GitHub repository
   --token              GitHub API token
   --token-file         GitHub API token file
   --help, -h           show help
   --version, -v        print the version
```

# Examples

Rebuild job `foo` for all opened pull requests:

```
LEEROY_USERNAME=user LEEROY_PASS=pass poule --repository "docker/docker" --token-file /home/icecrime/.github rebuild foo
```
