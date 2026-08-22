# Deterministic Java profile for bounded interview tasks.
FROM eclipse-temurin:21-jdk

RUN mkdir -p /opt/harness
COPY harness/run.sh /opt/harness/run.sh
RUN chmod 0555 /opt/harness/run.sh
WORKDIR /solution
