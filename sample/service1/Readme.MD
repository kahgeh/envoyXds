# Build
docker build --build-arg COMMIT_ID="$(git rev-parse HEAD)" . -t=service1

# Run
docker run -d --rm --name service1-i1 -P -e CONTAINER_NAME=service1-i1 service1
docker run -d --rm --name service1-i2 -P -e CONTAINER_NAME=service1-i2 service1