# Ensure docker api version is correctly set

Run `docker version`, and copy the value for `API Version`

`export DOCKER_API_VERSION="<version>"`

or 

in vscode launch.json set the env variables section 
```
            "env": {
                ...
                "DOCKER_API_VERSION": "<version>"
                ...
            },
```