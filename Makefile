DOCKER_USERNAME?=mconf
REPOSITORY?=aws-batch-exporter
IMAGE_NAME=$(DOCKER_USERNAME)/$(REPOSITORY)

up:
	docker-compose up

docker-build:
	docker build -f Dockerfile -t $(IMAGE_NAME):latest .