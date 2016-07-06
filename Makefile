REPOSITORY=so0k
IMAGE=ecs-sample
VERSION=1.0.0

default: builddocker

unixsetup:
	go get github.com/tools/godep
	go get golang.org/x/sys/unix

buildgo:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o /ecs-sample ./cmd/ecs-sample

builddocker:
	docker build -t ${REPOSITORY}/${IMAGE}:dev -f ./dev.Dockerfile .
	docker run -t ${REPOSITORY}/${IMAGE}:dev /bin/true
	docker cp `docker ps -q -n=1`:/ecs-sample .
	docker rm `docker ps -q -n=1`
	chmod 755 ./ecs-sample
	docker build --rm=true --tag=${REPOSITORY}/${IMAGE}:${VERSION} -f alpine.Dockerfile .

push:
	docker push ${REPOSITORY}/${IMAGE}:${VERSION}

up: buildocker
