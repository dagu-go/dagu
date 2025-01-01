.. _Yaml Format:

Writing DAGs
===========

.. contents::
    :local:

Introduction
------------
Dagu uses YAML files to define Directed Acyclic Graphs (DAGs) for workflow orchestration. This document covers everything you need to know about writing DAG definitions, from basic usage to advanced features.

Core Concepts
------------
Before diving into specific features, let's understand the basic structure of a DAG file and how steps are defined.

Minimal Example
~~~~~~~~~~~~~~
A DAG with two steps:

.. code-block:: yaml

  steps:
    - name: step 1
      command: echo hello
    - name: step 2
      command: echo world
      depends:
        - step 1

Using a pipe:

.. code-block:: yaml

  steps:
    - name: step 1
      command: echo hello world | xargs echo

Specifying a shell:

.. code-block:: yaml

  steps:
    - name: step 1
      command: echo hello world | xargs echo
      shell: bash

Schema Definition
~~~~~~~~~~~~~~~~
We provide a JSON schema to validate DAG files and enable IDE auto-completion:

.. code-block:: yaml

  # yaml-language-server: $schema=https://raw.githubusercontent.com/dagu-org/dagu/main/schemas/dag.schema.json
  steps:
    - name: step 1
      command: echo hello

The schema is available at `dag.schema.json <https://github.com/dagu-org/dagu/blob/main/schemas/dag.schema.json>`_.

Working Directory
~~~~~~~~~~~~~~~
Control where each step executes:

.. code-block:: yaml

  steps:
    - name: step 1
      dir: /path/to/working/directory
      command: some command

Basic Features
-------------

Environment Variables
~~~~~~~~~~~~~~~~~~~
Define variables accessible throughout the DAG:

.. code-block:: yaml

  env:
    - SOME_DIR: ${HOME}/batch
    - SOME_FILE: ${SOME_DIR}/some_file 
  steps:
    - name: task
      dir: ${SOME_DIR}
      command: python main.py ${SOME_FILE}

Parameters
~~~~~~~~~~
Pass positional parameters to steps:

.. code-block:: yaml

  params: param1 param2
  steps:
    - name: parameterized task
      command: python main.py $1 $2

Named Parameters
~~~~~~~~~~~~~~
Use named parameters for better clarity:

.. code-block:: yaml

  params:
    - FOO: 1
    - BAR: "`echo 2`"
  steps:
    - name: named params task
      command: python main.py ${FOO} ${BAR}

Code Snippets
~~~~~~~~~~~~

Run shell script with `$SHELL`:

.. code-block:: yaml

  steps:
    - name: script step
      script: |
        cd /tmp
        echo "hello world" > hello
        cat hello

You can run arbitrary script with the `script` field. The script will be executed with the program specified in the `command` field. If `command` is not specified, the default shell will be used.

.. code-block:: yaml

  steps:
    - name: script step
      command: python
      script: |
        import os
        print(os.getcwd())

Output Handling
--------------

Capture Output
~~~~~~~~~~~~~
Store command output in variables:

.. code-block:: yaml

  steps:
    - name: capture
      command: "echo foo"
      output: FOO  # Will contain "foo"

Redirect Output
~~~~~~~~~~~~~
Send output to files:

.. code-block:: yaml

  steps:
    - name: redirect stdout
      command: "echo hello"
      stdout: "/tmp/hello"
    
    - name: redirect stderr
      command: "echo error message >&2"
      stderr: "/tmp/error.txt"

Conditional Execution
------------------

Preconditions
~~~~~~~~~~~~
Run steps only when conditions are met:

.. code-block:: yaml

  steps:
    - name: monthly task
      command: monthly.sh
      preconditions:
        - condition: "`date '+%d'`"
          expected: "01"

Continue on Failure
~~~~~~~~~~~~~~~~~
Control flow when conditions aren't met:

.. code-block:: yaml

  steps:
    - name: optional task
      command: task.sh
      preconditions:
        - condition: "`date '+%d'`"
          expected: "01"
      continueOn:
        skipped: true

Scheduling
---------

Basic Scheduling
~~~~~~~~~~~~~~
Use cron expressions to schedule DAGs:

.. code-block:: yaml

  schedule: "5 4 * * *"  # Run at 04:05
  steps:
    - name: scheduled job
      command: job.sh

Skip Redundant Runs
~~~~~~~~~~~~~~~~~
Prevent unnecessary executions:

.. code-block:: yaml

    name: Daily Data Processing
    schedule: "0 */4 * * *"    
    skipIfSuccessful: true     
    steps:
      - name: extract
        command: extract_data.sh
      - name: transform
        command: transform_data.sh
        depends:
          - extract
      - name: load
        command: load_data.sh
        depends:
          - transform

When ``skipIfSuccessful`` is ``true``, Dagu checks if there's already been a successful run since the last scheduled time. If yes, it skips the execution. This is useful for:

- Resource-intensive tasks
- Data processing jobs that shouldn't run twice
- Tasks that are expensive to run

Note: Manual triggers always execute regardless of this setting.

Example timeline:
- Schedule: Every 4 hours (00:00, 04:00, 08:00, ...)
- At 04:00: Runs successfully
- At 05:00: Manual trigger → Runs (manual triggers always run)
- At 06:00: Schedule trigger → Skips (already succeeded since 04:00)
- At 08:00: Schedule trigger → Runs (new schedule window)

Retry Policies
~~~~~~~~~~~~
Automatically retry failed steps:

.. code-block:: yaml

  steps:
    - name: retryable task
      command: main.sh
      retryPolicy:
        limit: 3
        intervalSec: 5

Advanced Features
---------------

Running Sub-DAGs
~~~~~~~~~~~~~~
Organize complex workflows using sub-DAGs:

.. code-block:: yaml

  steps:
    - name: sub workflow
      run: sub_dag.yaml
      params: "FOO=BAR"

Command Substitution
~~~~~~~~~~~~~~~~~
Use command output in configurations:

.. code-block:: yaml

  env:
    TODAY: "`date '+%Y%m%d'`"
  steps:
    - name: use date
      command: "echo hello, today is ${TODAY}"

Lifecycle Hooks
~~~~~~~~~~~~~
React to DAG state changes:

.. code-block:: yaml

  handlerOn:
    success:
      command: echo "succeeded!"
    cancel:
      command: echo "cancelled!"
    failure:
      command: echo "failed!"
    exit:
      command: echo "exited!"
  steps:
    - name: main task
      command: echo hello

Repeat Steps
~~~~~~~~~~
Execute steps periodically:

.. code-block:: yaml

  steps:
    - name: repeating task
      command: main.sh
      repeatPolicy:
        repeat: true
        intervalSec: 60

User Defined Functions
~~~~~~~~~~~~~~~~~~~
Create reusable task templates:

.. code-block:: yaml

  functions:
    - name: my_function
      params: param1 param2
      command: python main.py $param1 $param2

  steps:
    - name: use function
      call:
        function: my_function
        args:
          param1: 1
          param2: 2

Field Reference
-------------

Quick Reference
~~~~~~~~~~~~~
Common fields you'll use most often:

- ``name``: DAG name
- ``schedule``: Cron schedule
- ``steps``: Task definitions
- ``depends``: Step dependencies
- ``skipIfSuccessful``: Skip redundant runs
- ``env``: Environment variables
- ``retryPolicy``: Retry configuration

DAG Fields
~~~~~~~~~
Complete list of DAG-level configuration options:

- ``name``: The name of the DAG (optional, defaults to filename)
- ``description``: Brief description of the DAG
- ``schedule``: Cron expression for scheduling
- ``skipIfSuccessful``: Skip if already succeeded since last schedule time (default: false)
- ``group``: Optional grouping for organization
- ``tags``: Comma-separated categorization tags
- ``env``: Environment variables
- ``logDir``: Output directory (default: ${HOME}/.local/share/logs)
- ``restartWaitSec``: Seconds to wait before restart
- ``histRetentionDays``: Days to keep execution history
- ``timeoutSec``: DAG timeout in seconds
- ``delaySec``: Delay between steps
- ``maxActiveRuns``: Maximum parallel steps
- ``params``: Default parameters
- ``preconditions``: DAG-level conditions
- ``mailOn``: Email notification settings
- ``MaxCleanUpTimeSec``: Cleanup timeout
- ``handlerOn``: Lifecycle event handlers
- ``steps``: List of steps to execute

Example DAG configuration:

.. code-block:: yaml

    name: DAG name
    description: run a DAG               
    schedule: "0 * * * *"                
    group: DailyJobs                     
    tags: example                        
    env:                                 
      - LOG_DIR: ${HOME}/logs
      - PATH: /usr/local/bin:${PATH}
    logDir: ${LOG_DIR}                   
    restartWaitSec: 60                   
    histRetentionDays: 3
    timeoutSec: 3600
    delaySec: 1                          
    maxActiveRuns: 1                     
    params: param1 param2                
    preconditions:                       
      - condition: "`echo $2`"           
        expected: "param2"               
    mailOn:
      failure: true                      
      success: true                      
    MaxCleanUpTimeSec: 300               
    handlerOn:                           
      success:
        command: echo "succeed"          
      failure:
        command: echo "failed"           
      cancel:
        command: echo "canceled"         
      exit:
        command: echo "finished"         

Step Fields
~~~~~~~~~
Configuration options available for individual steps:

- ``name``: Step name (required)
- ``description``: Step description
- ``dir``: Working directory
- ``command``: Command to execute
- ``stdout``: Standard output file
- ``output``: Output variable name
- ``script``: Inline script content
- ``signalOnStop``: Stop signal (e.g., SIGINT)
- ``mailOn``: Step-level notifications
- ``continueOn``: Failure handling
- ``retryPolicy``: Retry configuration
- ``repeatPolicy``: Repeat configuration
- ``preconditions``: Step conditions
- ``depends``: Dependencies
- ``run``: Sub-DAG reference
- ``params``: Sub-DAG parameters

Example step configuration:

.. code-block:: yaml

    steps:
      - name: complete example                  
        description: demonstrates all fields           
        dir: ${HOME}/logs                
        command: bash                    
        stdout: /tmp/outfile
        output: RESULT_VARIABLE
        script: |
          echo "any script"
        signalOnStop: "SIGINT"           
        mailOn:
          failure: true                  
          success: true                  
        continueOn:
          failure: true                  
          skipped: true                  
        retryPolicy:                     
          limit: 2                       
          intervalSec: 5                 
        repeatPolicy:                    
          repeat: true                   
          intervalSec: 60                
        preconditions:                   
          - condition: "`echo $1`"       
            expected: "param1"
        depends:
          - other_step_name
        run: sub_dag
        params: "FOO=BAR"

Global Configuration
------------------
Common settings can be shared using ``$HOME/.config/dagu/base.yaml``. This is useful for setting default values for:
- ``logDir``
- ``env``
- Email settings
- Other organizational defaults