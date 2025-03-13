FROM 416670754337.dkr.ecr.eu-west-2.amazonaws.com/ci-golang-build-1.24:latest

ARG GOPRIVATEURL="github.com/companieshouse"
ARG GITDOMAIN="git@github.com:"
ARG BUILDDIR=/build
ARG ROOTDIR=/

ENV GOPRIVATE=$GOPRIVATEURL

RUN yum update \
&& yum install -y \
git \
openssh-clients \
bash

RUN git config --global url.${GITDOMAIN}.insteadOf "https://github.com/"
RUN go install golang.org/x/vuln/cmd/govulncheck@latest
RUN mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts

WORKDIR $BUILDDIR

RUN --mount=type=ssh

COPY . ./

RUN chmod +x ./docker_start.sh

COPY ./docker_start.sh .

RUN chmod 700 docker_start.sh

CMD ["./docker_start.sh"]

COPY assets ./assets