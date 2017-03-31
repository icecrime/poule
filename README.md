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

# Documentation

See http://poule.readthedocs.io/en/latest/.