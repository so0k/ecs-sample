FROM golang:1.6
ENV SRC_DIR /go/src/github.com/so0k/ecs-sample

COPY Makefile $SRC_DIR/Makefile
#COPY Godeps/Godeps.json $SRC_DIR/Godeps/Godeps.json

WORKDIR $SRC_DIR
# Restore not needed as dependencies are vendored in
# RUN godep restore

#setup still pre-loads Godeps for dep management
RUN make unixsetup

COPY . $SRC_DIR
RUN make buildgo
CMD ["/bin/bash"]
