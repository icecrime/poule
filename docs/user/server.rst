Server mode
===========

Main configuration
------------------

Listening for events
~~~~~~~~~~~~~~~~~~~~

Using GitHub webhooks
^^^^^^^^^^^^^^^^^^^^^

Poule can listen on HTTP for incoming GitHub webhooks. Under this mode, the repository's webhook
settings in GitHub must point to the publicly accessible URL of a poule server instance.

The following configuration elements are required:

  - The ``http_listen`` address.
  - The ``http_secret`` value which must correspond to the secret value specified in the repository
    configuration on GitHub.

Example configuration:

.. code-block:: yaml

  http_listen: ":80"
  http_secret: "S3CR3T"

  repositories:
    icecrime/poule: ""

Using NSQ
^^^^^^^^^

`NSQ <http://nsq.io/>`_ is a "realtime distributed messaging platform" which, in combination with
`crosbymichael/hooks <https://github.com/crosbymichael/hooks>`_, can be used to distribute GitHub
events. Relying on a message queue for this use case has several advantages:

  - Messages are persisted: events will be queued when poule is offline and will catch-up as soon as
    it gets back online.
  - A single webhook endpoint in the repository's settings in GitHub can fan out messages to a
    variety of listeners through the messaging infrastructure.

Configuring poule to listens on NSQ requires several configuration elements:

  1. The ``nsq_lookupd`` address.
  2. The ``nsq_channel`` to subscribe to.
  3. For each repository, the queue name to monitor.

Example configuration:

.. code-block:: yaml

  nsq_channel: "poule"
  nsq_lookupd: "127.0.0.1:4161"

  repositories:
    icecrime/poule: "hooks-poule"

Repository configuration
------------------------

The server-mode configuration can contain both infrastructure-level settings (such as the NSQ
configuration) and operations. However, having the entire configuration in a single file is
impratical when managing a large collection of repositories.

In server mode, poule will look for a special ``poule.yml`` file at the root of each configured
repository and load it as repository-specific configuration. This allows each individual repository
and group of maintainers to manage their own set of rules. Furthermore, this allows to keep the
central configuration private as it typically contains secret information.

Monitoring for updates
~~~~~~~~~~~~~~~~~~~~~~

Repository-specific configurations will be loaded at poule startup. However, poule also provides a
builtin ``poule-updater`` operation which looks for merged pull requests which either modify or add
the special ``poule.yml`` file at the root of the repository.

When configured to be triggered on a pull request closed event, the operation will auto-refresh the
configuration settings for the repository without having to restart the server. One possibily is to
add this operation in the main configuration, hence covering all repositories:

.. code-block:: yaml

  common_configuration:

    # Poule updater watches for merged pull requests which modify the `poule.yml` file at the root
    # of the repository, and takes these changes into account live.
    - triggers:
          pull_request:   [ closed ]
      operations:
          - type:         poule-updater
