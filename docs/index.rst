Poule documentation
===================

🐔 Poule helps you automate operations on GitHub issues and pull requests.

It allows implementing snippets of behavior (called operations) *once* and provides a way to invoke
them in three different contexts:

  1. As a one-time invocation, on the entire stock of GitHub items.
  2. As part of a batch job alongside multiple other operations.
  3. As part of a long-running daemon triggered by GitHub webhooks or scheduled.

The project was created to manage automation on the `Moby project <https://github.com/moby/moby/>`_.

Installation
------------

Poule has not graduated to 1.0, so we don't do binary releases yet. In the meantime:

- Use the `pre-built image from Docker Hub <https://hub.docker.com/r/icecrime/poule/>`_: the
  ``latest`` tag maps to the current state of the ``master`` branch, while individual tags exist for
  pre-releases (e.g., ``0.4.0``).

    ``docker pull icecrime/poule:latest``

- Build from source using with no other dependency but `Docker <https://www.docker.com>`_.

    ``docker build -t poule https://github.com/icecrime/poule.git``

User guide
----------

.. toctree::
  :maxdepth: 2

  user/intro
  user/operations
  user/server

Examples
--------

One-time operations
~~~~~~~~~~~~~~~~~~~

Use the ``label`` operation to add label ``bug`` to issues which title or body matches the strings
"panic" in repository ``icecrime/poule``:

.. code-block:: bash

  $ poule --repository icecrime/poule label --filter is:issue bug:panic

Use the ``random-assign`` operation to randomly assigns pull requests older than 2 weeks among 3
GitHub users in repository ``icecrime/poule``:

.. code-block:: bash

  $ poule --repository icecrime/poule random-assign --filter is:pr --filter age:2w user1 user2 user3

Batch mode
~~~~~~~~~~

A batch on repository ``icecrime/poule`` which combines both of the operations described above,
and can together be executed in a single command.

.. code-block:: bash

  $ cat poule-batch.yml
  repository: icecrime/poule

  operations:

    - type: random-assign
      filters:
        age: "2w"
        is:  "pr"
      settings:
        users: [ "user1", "user2", "user3" ]

    - type: label
      filters:
        is: "issue"
      settings:
        patterns:
          bug: [ "panic" ]

  $ poule batch poule-batch.yml

Server mode
~~~~~~~~~~~

A server configuration which listens on port 80 for incoming `GitHub webhooks <https://developer.github.com/webhooks/>`_.
It applies the ``label`` operation described above *live* as issues get edited, opened, or reopened.
It also randomly assigns pull requests older than 2 weeks on a daily basis.

.. code-block:: bash

  $ cat poule-server.yml
  http_listen: ":80"
  http_secret: "S3CR3T"

  repositories:
    icecrime/poule: ""

  common_configuration:

    - triggers:
        issues: [ edited, opened, reopened ]
      operations:
        - type: label
          settings:
            patterns:
              bug: [ "panic" ]

    - schedule: "@daily"
      operations:
        - type: random-assign
          filters:
            age: "2w"
            is:  "pr"
          settings:
            users: [ "user1", "user2", "user3" ]

  $ poule serve --config poule-server.yml

Contributing
------------

- Repository: https://github.com/icecrime/poule/
- Issue tracker: https://github.com/icecrime/poule/issues
