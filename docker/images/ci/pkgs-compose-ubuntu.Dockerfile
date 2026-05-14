FROM cruizba/ubuntu-dind:noble-28.5.2

ENV DOCKER_HOST=tcp://docker:2375

RUN apt-get update && apt-get install -y \
    openjdk-17-jre-headless \
    python3.12 \
    python3.12-venv \
    rake \
    nodejs \
    npm \
    protobuf-c-compiler

CMD [ "bash" ]