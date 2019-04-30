# Run docker 

```
docker run --name envoy --rm -d -p 80:80 -p 9901:9901 \
-v $PWD/certs:/etc/certs \
-v $PWD/envoy.yaml:/etc/envoy/envoy.yaml envoyproxy/envoy-dev 
```