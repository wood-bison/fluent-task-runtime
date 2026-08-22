# Official SDK image: the same outer ContainerExecutor contract can grade
# managed code without pretending a Node image is a general runtime.
FROM mcr.microsoft.com/dotnet/sdk:10.0

RUN mkdir -p /opt/harness
COPY harness/run.sh /opt/harness/run.sh

# /solution (learner files), /hidden-tests (server-only tests), and /output
# are bind-mounted by container.executor at runtime.
WORKDIR /solution
