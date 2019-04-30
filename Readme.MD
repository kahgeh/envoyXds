# Background

An attempt to learn inner workings of an envoy control plane, by coding one up

# Automatic front proxy route update when containers are started

A simple way to investigate this area is to implement a process that pulls in service registry type data from  of docker containers and uses that as control plan data

In order to run the test project, run the following
- this control plane code - `go server/main.go` ( port 18000 )
- a frontproxy - a docker container with sample/frontproxy/envoy.yaml settings ( port 18000 )
- a service - served by 2 sample/service1 docker contaners

This will allow you to successfully call service1 via the frontproxy
```
    curl localhost/api/service1
    -> { "hello from": "service1-i1" }
    curl localhost/api/service1
    -> { "hello from": "service1-i2" }
```


# Pre-requisites

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
