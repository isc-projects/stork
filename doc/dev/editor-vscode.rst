
Visual Studio Code
******************

Debugging System tests
======================

1. Use VSCode
2. Add the following to .vscode/launch.json:

.. code-block:: json

    {
        // Use IntelliSense to learn about possible attributes.
        // Hover to view descriptions of existing attributes.
        // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
        "version": "0.2.0",
        "configurations": [
            {
                "name": "Launch GO file (th)",
                "type": "go",
                "request": "launch",
                "mode": "debug",
                "program": "${file}"
            }
            {
                "name": "Python: System tests th",
                "type": "python",
                "request": "launch",
                "module": "pytest",
                "python": "${workspaceFolder}/tools/python/bin/python",
                "console": "integratedTerminal",
                "cwd": "${workspaceFolder}/tests/system",
                "args": [
                    "-s",
                    "-k",
                    "${selectedText}"
                ],
                "env": {
                    // "KEA_VER": "2.0.2-isc20220227221539",
                    // "KEA_PUBLIC_REPO": "public/isc/kea-2-0",
                    // "KEA_PREMIUM_REPO": "<your-cloudsmith-token-here>/isc/kea-2-0-prv"
                    // "CS_REPO_ACCESS_TOKEN": "<your-cloudsmith-token-here>"
                }
            }
        ]

    }


3. Select system test name in editor. See `tests/system/tests/test*.py` files or
   run `rake systemtest:list`.
4. Set breakpoints (by clicking left of the line numbers in editor).
5. Hit F5 to run the configuration. Test will be running in console.

Go environment in VSCode
========================

Adding the following to `settings.json` will enable couple cool features:

1. Test Explorer. You can click on Testing (triangular beaker) on the left panel
   to get a nice, interactive list of tests that can be run separately or in
   groups.

2. Runs gopls, which is a language server that enables many more advanced
   features. One of the probably most useful one is the ability to use F12 to
   jump to definitions.

3. Enable golinter running in the IDE.

4. Set goroot and gopath variables.


.. code-block:: json

   "gopls": {
        "build.directoryFilters": [
            "-",
            "+backend"
        ]
    },
    "go.testExplorer.enable": true,
    "go.disableConcurrentTests": true,
    "go.gotoSymbol.ignoreFolders": [
        "webui",
        ".env",
        "venv",
        ".gitlab",
        ".pkgs-build",
        "tools"
    ],
    "go.logging.level": "verbose",
    "go.testEnvVars": {
        "STORK_DATABASE_TRACE": "1",
        "STORK_DATABASE_MAINTENANCE_NAME": "storktest"
    },
    "go.goroot": "${workspaceFolder}/tools/golang/go",
    "go.gopath": "${workspaceFolder}/tools/golang/gopath",
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "-c=../../backend/.golangci.yml"
    ],
