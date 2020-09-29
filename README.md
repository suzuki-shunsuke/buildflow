# buildflow

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
      command: |
        {{- range .Tasks -}}
          {{- if eq .Config.Name "foo" -}}
            echo "{{ .Result.Command.Stdout }}"
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
In the field `when`, we can use the expression of [antonmedv/expr](https://github.com/antonmedv/expr), which is the Go's third party library for the expression evaluation engine.

ex.

```yaml
    when: "3 > 1"
```

We can refer to the other task result and the pull request meta information in the expression as the variables.

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
    when: Tasks[0].Result.File.Text contains "bar"
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
    when: PR.owner == "octocat"
```

### Dynamic tasks

We can define tasks with a loop dynamically.

```yaml
- name: build
  tasks:
  - name: echo {{.Item.Value}}
    command:
      command: |
        echo {{.Item.Value}}
    items:
    - foo
    - bar
```

As the field `items`, we can use a list or a string which is an expression of [antonmedv/expr](https://github.com/antonmedv/expr).

```yaml
    items: |
      PR.labels
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
# The list of phases.
# Phases are run not in parallel but sequentially.
phases:
# The phase name. This must be unique and static.
- name: init
  # the list of tasks.
  tasks:
  # The task name. The value is parsed by text/template.
  - name: foo
    # Either `command` or `read_file` is required.
    command:
      # <shell> <shell_options>... <command> is run
      # ex. /bin/sh -c "echo hello"
      # The default shell is `/bin/sh`.
      # When the shell isn't set, the default shell_options is `-c`, otherwise the default shell_options is nothing.
      shell: /bin/sh
      shell_options:
      - -c
      command: echo hello
    # The condition whether the task is run.
    # The default is true.
    # The value should be true or false or a string which is an expression of antonmedv/expr.
    when: true
    # The task names which this task depends on or an expression of antonmedv/expr.
    # If `when` is an expression, the evaluation result should be true or false.
    # This task would be run after the dependent tasks are finished.
    # If the value is an expression, this task isn't run until the evaluation result becomes true.
    # Whether this task can be run is evaluated everytime a running task is finished.
    # The default is no dependency.
    dependency:
    - bar
    # The dynamic tasks.
    # items should be a list or a map or an expression of antonmedv/expr.
    # If items is an expression, the evaluation result should be a list or a map.
    items:
    - 1
    - 2
    # The meta attributes of the task.
    # You can use this field freely.
    # You can refer to this field in expressions and text/template.
    meta:
      service: foo
  - name: bar
    # read a file.
    # This is used to refer to the content of the file in subsequent tasks.
    read_file:
      # The file path to be read
      path: foo.txt
  condition:
    # When the skip is true, the phase is skipped.
    # The value should be true or false or an expression of antonmedv/expr.
    # If this is an expression, the evaluation result should be true or false.
    # The default is false.
    skip: false
    # `exit` is evaluated when the phase is finished.
    # If the `exit` is true, the build is finished and subsequent phases aren't run.
    # The value should be true or false or an expression of antonmedv/expr.
    # If this is an expression, the evaluation result should be true or false.
    # The default is false.
    exit: false
```

### Configuration variables

- PR: [Response body of GitHub API: Get a pull request](https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#get-a-pull-request)
- Files: [Response body of GitHub API: List pull requests files](https://docs.github.com/en/free-pro-team@latest/rest/reference/pulls#list-pull-requests-files)
- Phases
- Tasks
- Task
- Item
  - Key
  - Value
- Util: utility functions

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
Phases:
  init: # phase name
  - Result:
      Status: succeeded # queue, failed, succeeded, running, skipped
      Command:
        ExitCode: 0
        Stdout: hello # command's standard output
        Stderr: "" # command's standard error output
        CombinedOutput: hello # command output (Stdout + Stderr)
      File:
        Text: foo # The content of the file
    config:
      name:
  ...
Tasks: # the tasks of the current phase
- name: init # the task name
...
Task: # the current task
  name: init
  ...
Item: # The item of the dynamic tasks.
  key: 0
  value: zoo
Util:
  labelNames: func(PR.labels) []string: return a list of pull request label names
  env: https://golang.org/pkg/os/#Getenv
  string:
    split: https://golang.org/pkg/strings/#Split
    trimSpace: https://golang.org/pkg/strings/#TrimSpace
```

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

## LICENSE

[MIT](LICENSE)
