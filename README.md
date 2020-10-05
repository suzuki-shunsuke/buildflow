# buildflow

[![Build Status](https://github.com/suzuki-shunsuke/buildflow/workflows/CI/badge.svg)](https://github.com/suzuki-shunsuke/buildflow/actions)
[![Test Coverage](https://api.codeclimate.com/v1/badges/a92115bcb5226aaed505/test_coverage)](https://codeclimate.com/github/suzuki-shunsuke/buildflow/test_coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/suzuki-shunsuke/buildflow)](https://goreportcard.com/report/github.com/suzuki-shunsuke/buildflow)
[![GitHub last commit](https://img.shields.io/github/last-commit/suzuki-shunsuke/buildflow.svg)](https://github.com/suzuki-shunsuke/buildflow)
[![License](http://img.shields.io/badge/license-mit-blue.svg?style=flat-square)](https://raw.githubusercontent.com/suzuki-shunsuke/buildflow/master/LICENSE)

CLI tool for powerful build pipeline

We can define the build pipeline with YAML configuration, then we can run the build pipeline by `buildflow run` command.

buildflow provides the following features.

* Define the task dependency
* Run the multiple tasks in parallel
* Change the build behavior with the meta information of the Pull Request
  * file paths updated at the pull request
  * Pull Request labels
  * etc
* Define the dynamic tasks
* etc

## Install

Download from [GitHub Releases](https://github.com/suzuki-shunsuke/buildflow/releases)

```
$ buildflow --version
buildflow version 0.1.0
```

## Getting Started

### Hello World

Generate the configuration file by `buildflow init`

```
$ buildflow init
```

.buildflow.yaml

```yaml
pr: false
parallelism: 1
phases:
- name: main
  tasks:
  - name: hello
    command:
      command: echo hello
```

Then run `buildflow run`.

```
$ buildflow run

==============
= Phase: main =
==============
21:48:29UTC | hello | + /bin/sh -c echo hello
21:48:29UTC | hello |
21:48:29UTC | hello | hello
21:48:29UTC | hello |

================
= Phase Result =
================
task: hello
status: succeeded
exit code: 0
start time: 2020-09-29T21:48:29Z
end time: 2020-09-29T21:48:29Z
duration: 4.110401ms
+ /bin/sh -c echo hello
hello
```

The task `name` and `command` are parsed by Go's [text/template](https://golang.org/pkg/text/template), and functions of [sprig](https://github.com/Masterminds/sprig) can be used.

```yaml
pr: false
parallelism: 1
phases:
- name: main
  tasks:
  - name: hello {{ env "USER" }}
    command:
      command: echo {{ env "USER" }}
```

### Run multiple tasks in parallel

```yaml
pr: false
parallelism: 1
phases:
- name: main
  tasks:
  - name: foo
    command:
      command: |
        sleep 3
        echo foo
  - name: bar
    command:
      command: |
        sleep 3
        echo bar
```

The current parallelism is `1`, so the task `foo` and `bar` are run one by one.
There is no dependency between `foo` and `bar`, so which task is run first is random.
When `parallelism` is removed, `foo` and `bar` are run in parallel.

### Define the task dependency

```yaml
phases:
- name: main
  tasks:
  - name: foo
    command:
      command: |
        sleep 3
        echo foo
  - name: bar
    command:
      command: |
        echo bar
    dependency: # task dependency
    - foo # this task is run after the task foo is finished
```

### Refer to the other task result

```yaml
phases:
- name: main
  tasks:
  - name: foo
    command:
      command: |
        echo foo
  - name: bar
    command:
      command: >
        {{- range .Tasks -}}
          {{- if eq .Name "foo" -}}
            echo "{{ .Stdout }}"
          {{ end -}}
        {{- end -}}
    dependency:
    - foo
```

The other task's result can be refered as the variable.
Note that the task `A` can refer to only tasks which the task `A` depends on or the previous phase's tasks.

### Define the condition which the task is run

```yaml
phases:
- name: main
  tasks:
  - name: foo
    command:
      command: |
        echo foo
  - name: bar
    command:
      command: |
        echo bar
    when: false
```

The task `bar` isn't run because the condition `when` is `false`.
The type of `when` should be either boolean or [tengo](https://github.com/d5/tengo) script.
If `when` is a tengo script, the variable `result` should be defined and be boolean.

ex.

```yaml
    when: |
      result := 3 > 1
```

We can refer to the other task result and the pull request meta information in tengo scripts as the variables.

For the detail, please see [Configuraiton variables](#configuration-variables).

### Read a file in a task instead of executing a command

```yaml
phases:
- name: main
  tasks:
  - name: foo
    # read a file `foo.txt`
    # This is used to refer to the content of the file in subsequent tasks.
    read_file: 
      path: foo.txt
  - name: bar
    command:
      command: echo bar
    dependency:
    - foo
    when: |
      text := import("text")
      result := text.contains(Task[0].FileText, "dist")
```

### Refer to the pull request meta information in the configuration

When we use `buildflow` on the CI service such as CircleCI and GitHub Actions, we can refer to the pull request meta information.
Note that `buildflow` supports only GitHub as the source provider, and other providers such as GitLab and BitBucket aren't supported.
To refer to the pull request meta information, set the GitHub personal access token as the environment variable and set `true` to the configuration field `pr`.

```
$ export GITHUB_TOKEN=xxx # or GITHUB_ACCESS_TOKEN
```

```yaml
pr: true
phases:
...
```

To specify the pull request, the following information is required.

* repository owner name
* repository name
* pull request number

We can configure the owner and repository name in the configuration.

```yaml
pr: true
owner: suzuki-shunsuke
repo: buildflow
phases:
...
```

On the CI service such as CircleCI and GitHub Actions, `buildflow` gets the above information from the built-in environment variables automatically and get the pull request meta information.

* [GitHub API: Get a pull request](https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#get-a-pull-request)
* [GitHub API: List pull requests files](https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#list-pull-requests-files)

In the following example, the task is run only when the pull request author is `octocat`.

```yaml
    when: result := PR.owner == "octocat"
```

### Dynamic tasks

We can define tasks with a loop dynamically.

```yaml
- name: build
  tasks:
  - name: echo {{.Item.Key}}
    command:
      command: |
        echo {{.Item.Value}}
    items:
    - foo
    - bar
```

As the field `items`, we can use a list or a tengo script.
If `items` is a tengo script, the variable `result` should be defined and be a list.

```yaml
    items: |
      result := PR.labels
```

### Define multiple phases

```yaml
phases:
- name: init
  tasks:
  - name: init
    command:
      command: echo init
- name: main
  tasks:
  - name: main
    command:
      command: echo main
```

We can define the multiple phases.
Phases are run not in parallel but sequentially.
In the above configuration, at first the phase `init` is run and after that the task `main` is run.

### Skip phase

```yaml
phases:
- name: init
  tasks:
  - name: init
    command:
      command: echo init
  condition:
    skip: true # default is false
```

Like task's `when`, we can define the condition to skip the phase.
When the `phase.condtion.skip` is true, the phase is skipped.

### Exit build

```yaml
phases:
- name: init
  tasks:
  - name: init
    command:
      command: echo init
  condition:
    exit: true # default is false
```

`phase.condition.exit` is evaluated when the phase is finished.
If the `phase.condtion.exit` is true, the build is finished and subsequent phases aren't run.

## Configuration file path

The configuration file path can be specified with the `--config (-c)` option.
If the confgiuration file path isn't specified, the file named `.buildflow.yml` or `.buildflow.yaml` would be searched from the current directory to the root directory.

## Configuration Reference

```yaml
# when pr is true, buildflow gets the pull request meta information by GitHub API
# GitHub Personal Access Token is required.
# The default is false.
pr: true
# the repository owner name and repository name. This is used to get the pull request meta information.
# If pr is false, this is ignored.
# If this isn't set, buildflow tries to get the owner from the CI service's built-in environment variable.
owner: suzuki-shunsuke
repo: buildflow
# The maximum number of tasks which are run in parallel.
# The default is 0, which means there is no limitation.
parallelism: 1
# The meta attributes of the build.
# You can use this field freely.
# You can refer to this field in tengo scripts and text/template.
meta:
  service: foo
# The build condition
condtion:
  # When the skip is true, the build is skipped.
  # The value should be true or false or a tengo script.
  # If this is a tengo, the variable "result" should be defined and the type should be boolean.
  # The default is false.
  skip: false
  # When the fail is true, the build fails, which means the exit code of `buildflow run` isn't 0.
  # The value should be true or false or a tengo script.
  # If this is a tengo, the variable "result" should be defined and the type should be boolean.
  # By default `fail` is false if any phases failed.
  fail: false
# The list of phases.
# Phases are run not in parallel but sequentially.
phases:
# The phase name. This must be unique and static.
- name: init
  # The meta attributes of the phase.
  # You can use this field freely.
  # You can refer to this field in tengo scripts and text/template.
  meta:
    service: foo
  # the list of tasks.
  tasks:
  # The task name. The value is parsed by text/template.
  - name: foo
    # a tengo script which represents task's input.
    # The variable "result" should be defined.
    input: |
      result := {
        foo: "foo"
      }
    # Either `command` or `read_file` or `write_file` is required.
    command:
      # <shell> <shell_options>... <command> is run
      # ex. /bin/sh -c "echo hello"
      # The default shell is `/bin/sh`.
      # When the shell isn't set, the default shell_options is `-c`, otherwise the default shell_options is nothing.
      shell: /bin/sh
      shell_options:
      - -c
      # the command is executed where the configuration file exists.
      command: echo {{.Task.Input.foo}}
      # environment variables
      # In the environment variable name and value text/template can be used
      env:
        token: "{{ .Task.Name }}"
    # The condition whether the task is run.
    # The default is true.
    # The value should be true or false or a tengo script.
    # If this is a tengo, the variable "result" should be defined and the type should be boolean.
    when: true
    # The task names which this task depends on or a tengo script.
    # If `when` is a tengo script, the variable "result" should be defined and the type should be boolean.
    # This task would be run after the dependent tasks are finished.
    # If the value is a tengo script, this task isn't run until the evaluation result becomes true.
    # Whether this task can be run is evaluated everytime a running task is finished.
    # The default is no dependency.
    dependency:
    - bar
    # The dynamic tasks.
    # items should be a list or a map or a tengo script.
    # If items is a tengo script, the variable "result" should be defined and the type should be a list or a map.
    items:
    - 1
    - 2
    # The meta attributes of the task.
    # You can use this field freely.
    # You can refer to this field in tengo scripts and text/template.
    meta:
      service: foo
    # a tengo script which represents task's output.
    # This is useful to format task's result for subsequent tasks.
    # The variable "result" should be defined.
    output: |
      text := import("text")
      result := {
        foo: text.split(text.trim_space(Task.Stdout), "\n"),
      }
  - name: bar
    # read a file.
    # This is used to refer to the content of the file in subsequent tasks.
    read_file:
      # The file path to be read
      # If the path is the relative path, this is treated as the relative path from the directory where the configuration file exists.
      path: foo.txt
  - name: zoo
    # write a file.
    write_file:
      # The file path to be written
      # If the path is the relative path, this is treated as the relative path from the directory where the configuration file exists.
      path: foo.txt
      # The template of the file content.
      template: |
        {{ .Task.Name }}
  condition:
    # When the skip is true, the phase is skipped.
    # The value should be true or false or a tengo script.
    # If this is a tengo, the variable "result" should be defined and the type should be boolean.
    # The default is false.
    skip: false
    # `exit` is evaluated when the phase is finished.
    # If the `exit` is true, the build is finished and subsequent phases aren't run.
    # The value should be true or false or a tengo script.
    # If this is a tengo, the variable "result" should be defined and the type should be boolean.
    # The default is false.
    exit: false
    # `fail` is evaluated when the phase is finished.
    # If the `fail` is true, the phase fails.
    # The value should be true or false or a tengo script.
    # If this is a tengo, the variable "result" should be defined and the type should be boolean.
    # By default `fail` is false if any tasks failed.
    fail: false
```

### Configuration variables

- PR: [Response body of GitHub API: Get a pull request](https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#get-a-pull-request)
- Files: [Response body of GitHub API: List pull requests files](https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#list-pull-requests-files)
- Phases
- Phase
  - Name
- Tasks
- Task
- Item
  - Key
  - Value

#### Example

Express the variables as YAML.

```yaml
PR:
  url: https://api.github.com/repos/octocat/Hello-World/pulls/1347
  id: 1
  ...
Files:
- sha: bbcd538c8e72b8c175046e27cc8f907076331401
  filename: file1.txt
...
phases:
  init: # phase name
    Status: succeeded
    Tasks:
    - Name: foo
      Status: succeeded # queue, failed, succeeded, running, skipped
      ExitCode: 0
      Stdout: hello # command's standard output
      Stderr: "" # command's standard error output
      CombinedOutput: hello # command output (Stdout + Stderr)
      Meta:
        foo: foo
    - Name: bar
      Status: succeeded
      FileText: foo # The content of the file
  ...
Tasks: # the tasks of the current phase
- Name: init # the task name
...
Task: # the current task
  Name: init
  ...
Item: # The item of the dynamic tasks.
  Key: 0
  Value: zoo
```

### Custom Functions of template

* http://masterminds.github.io/sprig/
* LabelNames: func(PR.labels) []string: return a list of pull request label names
* GetTaskByName: func(tasks, name) task: get a task by task name

## Usage

```
$ buildflow help
NAME:
   buildflow - run builds. https://github.com/suzuki-shunsuke/buildflow

USAGE:
   buildflow [global options] command [command options] [arguments...]

VERSION:
   0.1.0

COMMANDS:
   run      run build
   init     generate a configuration file if it doesn't exist
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)
```

```
$ buildflow init --help
NAME:
   buildflow init - generate a configuration file if it doesn't exist

USAGE:
   buildflow init [command options] [arguments...]

OPTIONS:
   --help, -h  show help (default: false)
```

```
$ buildflow run --help
NAME:
   buildflow run - run build

USAGE:
   buildflow run [command options] [arguments...]

OPTIONS:
   --owner value             repository owner
   --repo value              repository name
   --github-token value      GitHub Access Token [$GITHUB_TOKEN, $GITHUB_ACCESS_TOKEN]
   --log-level value         log level
   --config value, -c value  configuration file path
   --help, -h                show help (default: false)
```

## Where to run the task

The task is run at not the current directory (`$PWD`) but the directory where the configuration file exists.

## LICENSE

[MIT](LICENSE)
