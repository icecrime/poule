Introduction
============

Synopsis
--------

::

  NAME:
    poule - Mass interact with GitHub issues & pull requests

  USAGE:
     poule [global options] command [command options] [arguments...]
  
  VERSION:
     0.4.0
  
  COMMANDS:
       batch     Run groups of commands described in files
       serve     Operate as a daemon listening on GitHub webhooks
       validate  Validate a Poule repository configuration file
       help, h   Shows a list of commands or help for one command
  
     Operations:
       ci-label-clean     Clean CI failure labels
       dco-check          Check DCO on pull requests
       label              Apply label(s) to items which title or body matches a pattern
       poule-updater      Update the poule configuration for the specified repository
       prune              Prune outdated issues
       random-assign      Assign items to a random username from the `users` list.
       rebuild            Rebuild configurations of a given state
       version-label      Apply version labels to issues
       version-milestone  Attach merged pull requests to the upcoming version's milestone
  
  GLOBAL OPTIONS:
     --debug, -D         enable debug logging
     --dry-run           simulate operations
     --repository value  GitHub repository
     --token value       GitHub API token [$POULE_GITHUB_TOKEN]
     --token-file value  GitHub API token file [$POULE_GITHUB_TOKEN_FILE]
     --help, -h          show help
     --version, -v       print the version

Global options
--------------

Specifying a GitHub API token
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

A GitHub API token must be provided for poule to execute any modifying action (such as labeling an
issue, or closing a pull request). The token can be specified:

  - Directly by providing its value through the ``--token`` flag or the ``$POULE_GITHUB_TOKEN``
    environment variable.
  - Indirectly by providing the path to a file containing a token through the ``--token-file`` flag
    or the ``$POULE_GITHUB_TOKEN_FILE`` environment variable.

Simulating execution
~~~~~~~~~~~~~~~~~~~~

When ``--dry-run`` is specified, poule retrieves GitHub issues and pull requests and calls
operations as it normally would but doesn't actually *apply* the operations. Each operation will log
as it is called, and what it would have done if applied.

Keep in mind that poule in dry run still issues the API calls necessary to retrieve GitHub data, and
as a result contributes to consuming the GitHub's user API limit.

Running operations
------------------

Poule is all about running :doc:`operations` on GitHub issues and pull requests. An operation is a
snippet of GitHub automation, such as adding a label to items which body matches a given string.
Once implemented, an operation can be reused in different contexts:

  1. As a one-time invocation, on the entire stock of GitHub items.
  2. As part of a batch job alongside multiple other operations.
  3. As part of a long-running daemon triggered by GitHub webhooks or scheduled.

One-time invocation
~~~~~~~~~~~~~~~~~~~

Each operation gets surfaced in the command-line as its own subcommand, making the invocation of a
one-off operation straightforward. All operations subcommand support the ``--filter`` flag which
allows to restrict the items on which the operation will be applied. Additionally, each operation
defines its own set of flags and its own input format: refer to the ``--help`` output for
operation-specific information.

Batch execution
~~~~~~~~~~~~~~~

In batch execution, a collection of operations is described in `YAML <http://yaml.org/>`_ format.
Similarly to the command-line invocation, each operation can be associated with a set of filters, as
well as operation-specific settings.

Server mode
~~~~~~~~~~~

This is of course the most interesting mode, and deserves as such an entire documentation page:
:doc:`server`.

Configuring execution
---------------------

Filtering
~~~~~~~~~

The following filter types are supported to restrict the set of items on which a given operation
should be applied:

+----------+--------------------------------------------+---------------------------------------+
| Type     | Passes if                                  | Values                                |
+==========+============================================+=======================================+
| age      | Creation date > value                      | E.g.,: ``2d``, ``3w``, ``4m``, ``1Y`` |
+----------+--------------------------------------------+---------------------------------------+
| assigned | Issue is assigned == value                 | ``true`` or ``false``                 |
+----------+--------------------------------------------+---------------------------------------+
| comments | # comments matches predicate               | E.g.,: ``"=0"``, ``">10"``, ``"<20"`` |
+----------+--------------------------------------------+---------------------------------------+
| labels   | All specified labels are set               | E.g.,: ``"label1,label2"``            |
+----------+--------------------------------------------+---------------------------------------+
| ~labels  | None of the specified labels are set       | E.g.,: ``"label1,label2"``            |
+----------+--------------------------------------------+---------------------------------------+
| is       | Type of item == value                      | ``pr`` or ``issues``                  |
+----------+--------------------------------------------+---------------------------------------+

All operations subcommands support the ``--filter`` with the following format::

  --filter <filter_type_1>:<filter_value_1> [--filter <filter_type_n>:<filter_value_n> ...]

When describing operation in YAML format (either for batch or server mode), filtering is defined as
a ``filters`` mapping filter types to their respective values::

  filters:
  	<filter_type_1>: <filter_value_1>
  	<filter_type_n>: <filter_value_n>

Note that sequences are used instead of comma separated values for the ``labels`` and ``~labels``
filters, for example::

   --filter is:issue --filter label:bug --filter age:2d

Is expressed in YAML as the following::

  filters:
    age:   2d
    is:    issue
    label: [ bug ]