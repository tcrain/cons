FROM golang:buster

WORKDIR  /

RUN apt-get update

RUN apt-get -y install git htop jq pkg-config autoconf automake libtool gcc libssl-dev python3-pytest unzip xsltproc doxygen graphviz make cmake ninja-build python3-pytest-xdist xsltproc sudo

ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib
ENV PKG_CONFIG_PATH=$PKG_CONFIG_PATH:$GOPATH/src/github.com/open-quantum-safe/liboqs-go/.config

RUN git clone -b master https://github.com/open-quantum-safe/liboqs.git

WORKDIR liboqs/build
RUN cmake -GNinja -DBUILD_SHARED_LIBS=ON ..
RUN ninja
RUN ninja install

# RUN go get github.com/go-delve/delve/cmd/dlv

WORKDIR $GOPATH/src/github.com/open-quantum-safe/
RUN git clone https://github.com/open-quantum-safe/liboqs-go
