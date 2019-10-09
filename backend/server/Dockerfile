FROM ubuntu:19.10
WORKDIR /server
COPY main /server/main
#RUN apt update && apt install -y --no-install-recommends python3 python3-pytest git openssh-client ca-certificates
ENV SERVER localhost
CMD  /server/main --port 5000 --host 0.0.0.0
