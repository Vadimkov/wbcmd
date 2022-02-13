#build stage
FROM ubuntu:21.10
RUN apt update -y && apt install mosquitto mosquitto-clients golang -y
