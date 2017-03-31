Operations
==========

Definition
----------

An operation is a snippet of GitHub automation, for example: adding a label, closing a pull request,
or commenting on an issue.

  - Operations are idempotent, which means that they can safely be applied multiple times.
  - An operation can apply to GitHub issues, pull requests, or both. For example, a ``label``
    operation may know to operate independently on issues and pull requests, while a ``rebuild``
    operation which triggers CI may only apply on pull requests.
  - A :doc:`catalog` of builtin operations is provided and documented.

Builtin operations
------------------

+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| Operation             | Docker specific | Issues | Pull Requests  | Purpose                                                               |
+=======================+=================+========+================+=======================================================================+
| ``ci-label-clean``    |                 |        | ☑              | Remove CI failures labels where necessary.                            |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``dco-check``         | 🐳.             |        | ☑              | Check for commit signatures, label and post a comment if missing.     |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``label``             |                 | ☑      | ☑              | Auto-label issues and pull requests according on matching regexps.    |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``poule-updater``     |                 |        | ☑              | Reload ``poule`` configuration when a pull request modifies it.       |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``prune``             |                 | ☑      |                | Manage issues with no activities.                                     |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``random-assign``     |                 | ☑      | ☑              | Auto-assign a random user to issues and pull requests.                |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``rebuild``           | 🐳              |        | ☑              | Rebuild all or selected pull request jobs.                            |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``version-label``     | 🐳              | ☑      |                | Add a ``version/x`` label based on Docker version string in the body. |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+
| ``version-milestone`` | ~               |        | ☑              | Add merged pull requests to the upcoming milestone.                   |
+-----------------------+-----------------+--------+----------------+-----------------------------------------------------------------------+

More details on each operation can be found on `GitHub <https://github.com/icecrime/poule/blob/master/src/poule/operations/catalog/README.md>`_.

Creating custom operations
--------------------------

Creating custom operations is not yet supported and requires modifying the project. However, issue
`icecrime/poule#4 <https://github.com/icecrime/poule/issues/4>`_ is about adding support for Golang
1.8 plugins in such way that custom operations can be added at runtime.