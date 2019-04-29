# Build
docker build --build-arg COMMIT_ID="$(git rev-parse HEAD)" . -t=service1

# Run
docker run -d --rm --name service1 -P service1