# Background

An attempt to learn inner workings of an envoy control plane, by coding one up

# Service Discovery to provide control plane data

A simple and complete way to investigate this area is to implement a service discovery of docker containers. 

In order to run the test project, run the following
- this control plane code - `go server/main.go` ( port 18000 )
- a frontproxy - a docker container with sample/frontproxy/envoy.yaml settings ( port 18000 )
- a service - sample/service1 docker contaner

This will allow you to successfully call service1 via the frontproxy
    `curl localhost/api/service1`

