.. _Configuration Options:

Configurations
==============

.. contents::
    :local:

.. _Environment Variables:

Environment Variables
----------------------

The following environment variables can be used to configure the Dagu. Default values are provided in the parentheses:

- ``DAGU_HOST`` (``127.0.0.1``): The host to bind the server to.
- ``DAGU_PORT`` (``8080``): The port to bind the server to.
- ``DAGU_DAGS`` (``$HOME/.config/dagu/dags``): The directory containing the DAGs.
- ``DAGU_IS_BASICAUTH`` (``0``): Set to 1 to enable basic authentication.
- ``DAGU_BASICAUTH_USERNAME`` (``""``): The username to use for basic authentication.
- ``DAGU_BASICAUTH_PASSWORD`` (``""``): The password to use for basic authentication.
- ``DAGU_LOG_DIR`` (``$HOME/.local/share/logs``): The directory where logs will be stored.
- ``DAGU_DATA_DIR`` (``$HOME/.local/share/history``): The directory where application data will be stored.
- ``DAGU_SUSPEND_FLAGS_DIR`` (``$HOME/.config/dagu/suspend``): The directory containing DAG suspend flags.
- ``DAGU_ADMIN_LOG_DIR`` (``$HOME/.local/share/admin``): The directory where admin logs will be stored.
- ``DAGU_BASE_CONFIG`` (``$HOME/.config/dagu/base.yaml``): The path to the base configuration file.
- ``DAGU_NAVBAR_COLOR`` (``""``): The color to use for the navigation bar. E.g., ``red`` or ``#ff0000``.
- ``DAGU_NAVBAR_TITLE`` (``Dagu``): The title to display in the navigation bar. E.g., ``Dagu - PROD`` or ``Dagu - DEV``
- ``DAGU_WORK_DIR``: The working directory for DAGs. If not set, the default value is DAG location. Also you can set the working directory for each DAG steps in the DAG configuration file. For more information, see :ref:`specifying working dir`.
- ``DAGU_CERT_FILE``: The path to the SSL certificate file.
- ``DAGU_KEY_FILE`` : The path to the SSL key file.
- ``DAGU_TZ`` (``""``): The timezone to use for the server. By default, the server will use the system's local timezone.

Config File
--------------

You can create ``admin.yaml`` file in ``$HOME/.config/dagu/`` to override the default configuration values. The following configuration options are available:

.. code-block:: yaml

    host: <hostname for web UI address>                          # default: 127.0.0.1
    port: <port number for web UI address>                       # default: 8080

    # to show latest status of dags from today or history
    latestStatusToday: true

    # path to the DAGs directory
    dags: <the location of DAG configuration files>              # default: ${HOME}/.config/dagu/dags
    
    # Web UI Color & Title
    navbarColor: <ui header color>                               # header color for web UI (e.g. "#ff0000")
    navbarTitle: <ui title text>                                 # header title for web UI (e.g. "PROD")
    
    # Basic Auth
    isBasicAuth: <true|false>                                    # enables basic auth
    basicAuthUsername: <username for basic auth of web UI>       # basic auth user
    basicAuthPassword: <password for basic auth of web UI>       # basic auth password

    # API Token
    isAuthToken: <true|false>                                    # enables API token
    authToken: <token for API access>                            # API token

    # Base Config
    baseConfig: <base DAG config path>                           # default: ${HOME}/.config/dagu/base.yaml

    # Working Directory
    workDir: <working directory for DAGs>                        # default: DAG location

    # SSL Configuration
    tls:
        certFile: <path to SSL certificate file>
        keyFile: <path to SSL key file>
    
    # Timezone Configuration
    tz: <timezone>                                               # default: "" (e.g. "Asia/Tokyo")

.. _Host and Port Configuration:

Server's Host and Port Configuration
-------------------------------------

To specify the host and port for running the Dagu server, there are a couple of ways to do it.

The first way is to specify the ``DAGU_HOST`` and ``DAGU_PORT`` environment variables. For example, you could run the following command:

.. code-block:: sh

    DAGU_PORT=8000 dagu server

The second way is to use the ``--host`` and ``--port`` options when running the ``dagu server`` command. For example:

.. code-block:: sh

    dagu server --port=8000

See :ref:`Environment Variables` for more information.
