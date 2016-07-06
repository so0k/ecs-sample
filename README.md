## Overview

In this sample Repo we will use Docker to quickly and easily get started with and scale a Real-Time messaging app written in Golang.

![Screenshot of app](docs/images/app_screenshot.png)

As a reference, a very lightweight app from the [Toptal: Going Real-Time with Redis Pub/Sub](https://www.toptal.com/go/going-real-time-with-redis-pubsub) article was used. The code has slightly been modified to better demonstrate Docker related concepts and is fully included in this repository.

Changes to the original application:

- Added additional logging
- Added indicator which application container is serving the client
- Added [exponential back-off](https://medium.com/@kelseyhightower/12-fractured-apps-1080c73d481c#.zbqpolxwo) as a best practice for applications running in the cloud.

The application allows users to upload images and see real time comments on those images. Clicking the image will show indicators where the image was clicked for every user. All this functionality was written by the Toptal developer.

To implement the above functionality, the following stack will be used:

- AWS S3: To store the user-uploaded images.
- MongoDB: As a Document Oriented Database keeping track of images stored on S3 and the comments of users.
- Redis Pub/Sub: Redis as a Publish/Subscribe messaging system to propagate real time updates
- App: the Golang application to serve the webpage and manage the websockets with client browsers
- Nginx: As a load balancer to easily scale the application horizontally.

**Note**: Nginx is used as a load balancer while running the full stack locally, in a production environment a more robust load balancer setup should be considered.

*Note* IAM permissions for EC2, ECS and CloudFormation are required to follow along this guide.

Development environment:

 - Tested on OSX El Capitan with Bash
 - Install [Docker For Mac](https://beta.docker.com/)
 - Ensure you have a working AWS Account (we will do sample S3 setup as part of these instructions)
 - Download [jq](https://github.com/stedolan/jq/releases) to easily work with AWS resources from the CLI

## Table of Contents:

 - [Setting up S3](#setting-up-s3) : Set up all necessary objects on AWS using the CLI.
 - [Getting to know Docker for Mac](#getting-to-know-docker-for-mac): Ensuring Docker is working as expected.
 - [Getting to know Redis](#getting-to-know-redis): Using Docker to play with Redis - no installs required!
 - [Getting to know MongoDB](#getting-to-know-mongodb): Using Docker to play with MongoDB - no installs required!
 - [Playing with the full application stack](#playing-with-the-full-application-stack): `git clone` + `make` + `docker-compose` to stand up the full app locally.
 - [Understanding Container Distribution](#understanding-container-distribution): How do we take containers to the cloud?
 - [Amazon ECS Introduction](#amazon-ecs-introduction): High level overview of ECS (To be completed)
 - [Deploying to Amazon ECS](#deploying-to-amazon-ecs): Using AWS CLI to deploy full app to ECS (To be completed)

## Setting up S3

Install AWS-CLI (You will need access keys to use the cli)
```
pip install awscli
aws configure
```

Ensure jq is working properly:
```console
jq --version
```

Expected Output (similar to this):
```console
jq-1.5
```

Create an account to give S3 access to the Application (Don't use root account)
```console
aws iam create-user --user-name sample_app
```

Create Access Key and save to `.env` file:
```console
aws iam create-access-key --user-name sample_app | jq -r '"AWS_ACCESS_KEY_ID=\(.AccessKey.AccessKeyId)","AWS_SECRET_ACCESS_KEY=\(.AccessKey.SecretAccessKey)"' >> .env
```

Create S3 Bucket (You will need to change instructions according to your bucket name, samples here use `ecs-sample`)
```console
aws s3 mb s3://ecs-sample --region ap-southeast-1
```

Add your S3 bucket name to your `.env` file:
```console
echo "S3_BUCKET_NAME=ecs-sample" >> .env
```

Create Policy Document for S3 Bucket
```console
cat - << EOF > SampleAppS3Policy.json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "s3:*",
            "Resource": [
                "arn:aws:s3:::ecs-sample/",
                "arn:aws:s3:::ecs-sample/*"
            ]
        }
    ]
}
EOF
```

Upload Policy Document
```console
policy_arn=`aws iam create-policy --policy-name MyAppS3Access --description "Give S3 Access to ecs-sample bucket" --policy-document file://SampleAppS3Policy.json | jq -r '.Policy.Arn'`
```

Attach Policy Document to `sample_app` user
```console
aws iam attach-user-policy --user-name sample_app --policy-arn $policy_arn
```

**Note**: CloudFormation should be considered to automate the above steps.

## Getting to know Docker for Mac

Once Docker for Mac has been installed and is running, confirm everything is installed correctly:

```console
docker version
```

Expected output (similar to this)
```console
Client:
 Version:      1.11.2
 API version:  1.23
 Go version:   go1.5.4
 Git commit:   b9f10c9
 Built:        Wed Jun  1 21:20:08 2016
 OS/Arch:      darwin/amd64

Server:
 Version:      1.11.2
 API version:  1.23
 Go version:   go1.5.4
 Git commit:   56888bf
 Built:        Mon Jun  6 23:57:32 2016
 OS/Arch:      linux/amd64
```

```console
docker-compose version
```

Expected output (similar to this)
```console
docker-compose version 1.7.1, build 0a9ab35
docker-py version: 1.8.1
CPython version: 2.7.9
OpenSSL version: OpenSSL 1.0.1j 15 Oct 2014
```

docker "Hello world!"

```console
docker run hello-world
```

Expected output:
```console
...
a9d36faac0fe: Pull complete
Digest: sha256:e52be8ffeeb1f374f440893189cd32f44cb166650e7ab185fa7735b7dc48d619
Status: Downloaded newer image for hello-world:latest

Hello from Docker.
This message shows that your installation appears to be working correctly.

To generate this message, Docker took the following steps:
 1. The Docker client contacted the Docker daemon.
 2. The Docker daemon pulled the "hello-world" image from the Docker Hub.
 3. The Docker daemon created a new container from that image which runs the
    executable that produces the output you are currently reading.
 4. The Docker daemon streamed that output to the Docker client, which sent it
    to your terminal.

To try something more ambitious, you can run an Ubuntu container with:
 $ docker run -it ubuntu bash

Share images, automate workflows, and more with a free Docker Hub account:
 https://hub.docker.com

For more examples and ideas, visit:
 https://docs.docker.com/engine/userguide/
```

At this point, you may read the [userguide](https://docs.docker.com/engine/userguide/) linked above. Concepts will be explained as they are encountered in this guide as well.

## Getting to know Redis

Once Docker is installed, you will never have to install packages on your machine to play with interesting technology again. You can simple run the service in a container and remove every trace of it when done.

First, lets spin up a Daemonized (`-d`) redis container (named `redis-test`):
```console
docker run -d --name redis-test redis:3.2.0-alpine
```

Verify the container is running:
```console
docker ps
```

To play with this redis, we need the `redis-cli`, but we do not need to install it on our machine as `redis-cli` is bundled in the redis container!

Get a shell (`-it`) in a 2nd redis container linked (`--link`) to the first:
```console
docker run -it --rm --link redis-test redis:3.2.0-alpine /bin/sh
```

From within this container, connect to redis server (`-h redis-test`):
```console
redis-cli -h redis-test
```

Test Redis
```console
SET lives 9
INCR lives
GET lives
```

Let's play with Pub/Sub features of Redis:
```console
SUBSCRIBE channel
```

Launch a 2nd redis container (re-use the exact same command to launch the container from above in a separate terminal)
```console
redis-cli -h redis-test
```

Publish data from 2nd container to `channel`
```console
PUBLISH channel "hello from container2"
```

You should see the message broadcasted to all subscribed clients. Notice that once a connection is in subscription mode, you can no longer use it to send messages on. To both send and receive, 2 separate connections to the redis server are required.

## Getting to know MongoDB

Very similar to the Redis experiment above, quickly launch a MongoDB server:

Launch a Daemonised Mongo container:
```console
docker run -d --name mongo-test mongo:3.2.6
```

Launch a container to play with the mongo shell:
```console
docker run -it --rm --link mongo-test mongo:3.2.6 /bin/bash
```

Connect via Mongo shell
```console
mongo mongo-test:27017
```

Insert sample documents
```javascript
db.messages.insert(
{
    "message": "hello",
    "sender": "me"
})

db.messages.insert(
{
    "message": "world",
    "sender":"you"
})
```

Select sample messages
```javascript
db.messages.find()
```

Select sample messages with a conditions document
```javsacript
db.messages.find( { "sender": "you" })
```

Create an ascending index on `sender` field of the messages collection
```javascript
db.messages.createIndex({"sender": 1})
```

## Playing with the full application stack

The application is written in Golang. all dependencies have been vendored in with `Godeps`. However, to play with the application, golang does not have to be installed locally. Everything is handled through Docker.

Let's first clean up the `redis-test` and `mongo-test` containers:
```console
docker stop redis-test mongo-test && docker rm redis-test mongo-test
```

Clone the application:
```console
git clone https://github.com/so0k/ecs-sample.git
```

**Note**: Once you are required to develop further, setting up a local golang environment and cloning the application under the correct path is still easy and possible.

Build the application (using Docker)
```console
make
```
**Note**: This makefile is inspired by [Nicola Paolucci's article](https://developer.atlassian.com/blog/2015/07/osx-static-golang-binaries-with-docker/).

The full application stack is defined in a declarative [docker-compose.yaml](docker-compose.yaml) file at the root of this repository.

The Environment configuration for our application is stored in the `.env` file we have incrementally been creating in the above setup steps. Docker Compose will pass all these parameters from the `.env` file to our application via ENVIRONMENT VARIABLES.

Two parameters are still missing, add these as follows:
```console
echo "MONGO_URL=mongodb://mongo/ecs-sample" >> .env
echo "REDIS_URL=redis://redis" >> .env
```
*Note*: We have defined the MongoDB hostname as `mongo` and the Redis hostname as `redis` in the docker-compose file.

Let's `watch` running containers with the following command in a separate terminal:
```console
watch -n 1 "docker ps --format='table{{.Image}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.ID}}'"
```

We are now ready to stand up the application stack:
```console
docker-compose up -d
```

Once all containers are running, you should be able to open [localhost:80](http://localhost:80/)

Try to scale the application:
```console
docker-compose scale app=5
```

Opening multiple browsers should demonstrate that client sessions are load balanced to separate application servers behind the load balancers and that all real-time events are propagated across the cluster.

Get a shell on one of the running application containers:
```console
docker exec -it ecssample_app_1 /bin/sh
```

Review the DNS records published by the Docker Engine:
```console
nslookup apps
```

We can ping `mongo` and `redis` hosts from app container:
```console
ping mongo
ping redis
```

Our `mongo` and `redis` containers are isolated from the load balancer:

Get a shell on the running load balancer.
```console
docker exec -it lb /bin/sh
```

We can ping apps:
```console
ping apps
```

We can not ping `mongo` or `redis` hosts
```console
ping mongo
ping redis
```

Clean up
```console
docker-compose stop
docker-compose rm
```

Notes:

- The application uses DNS round robin for the nginx load balancer to test scaling locally
- A `so0k/ecs-sample:dev` container is available to play with the application source code

Run the Dev container as follows:
```console
docker run -it --name ecs-sample-dev -v $PWD:/go/src/github.com/so0k/ecs-sample so0k/ecs-sample:dev
```

## Understanding Container Distribution

The container images built as part of this demonstration are currently only available on our host. An important component of the container
eco-system is having the ability to ship these images to different hosts.

Similar to the concept of Software Repositories, container image repositories are designed for the purpose of delivering container
images to hosts. In the Docker ecosystem, distribution is managed through a Registry service.  Several Registry service offerings are available:

1. *Docker Hub*: This is the public Registry provided by Docker and the default registry used by every Docker client.
   The `mongo`, `redis` and `nginx` images we ran earlier were all obtained from the Docker Hub.
   Public Repositories on the Docker Hub are free (Similar to Public Repositories on GitHub).

2. *Self-Hosted Registry*: An open source version of the [Registry](http://github.com/docker/distribution) is available on GitHub.
   This allows anyone to host Docker Images privately, with the overhead of configuring and maintaining the service.

3. *Amazon ECR*: [Amazon ECR](http://aws.amazon.com/ecr/) is integrated with Amazon ECS and eliminates the need to operate
   your own container repositories or  worry about scaling the underlying infrastructure.
   Integration with IAM provides resource-level control of each repository. You pay only for the amount of data you store
   in your repositories and data transferred to the internet.

For this sample application, a public repository on the Docker Hub was used, following these steps:

1.   Create a Docker Hub account by [signing up](https://hub.docker.com/). Similar to GitHub, credentials were not required to pull images, however they are required to push images.
2.   Provide your docker client with your account credentials:

     ```console
     docker login
     ```

3.   Review the repository and image name in the Makefile provided with this repository (change to match your Docker Hub  account and rebuild image if needed)
4.   You may use the `make push` target to tag and push the container image to the Docker Hub:

     ```console
     make push
     ```

*Note* Changing the Repository and Image name in the Makefile will also require you to revise the `docker-compose.yaml`.
The changes required to this file are not covered in the current version of this guide, PR's are welcome.

## Docker 1.12 Introduction

### Using Docker For Mac:

```
docker swarm init
docker service create ...
```

### Deploying 3 node cluster on EC2

#### Provision the AWS infrastructure Using terraform

Create a `terraform.tfvars` file at the root of the directory with you AWS keys

Sample file:
```
access_key = "<SAMPLE>"
secret_key = "<SAMPLE>"

public_key_path = "~/.ssh/id_rsa.pub"
key_name = "ecs-sample"
```

Review the infrastructure defined in `docker-cluster.tf` of this repository:
```
terraform plan
```

Create the cluster on AWS:
```
terraform apply
```

After it completes, it should have returned a comma separated list of the nodes.

You may extract this list again from the local terraform state as follows:

```
terraform output nodes
```

#### Create the Docker 1.12 Cluster

ssh to the nodes and let the engines form a swarm

```
ssh ubuntu@<first-node-ip>
sudo -i
docker swarm init
```
Note private IP of Leader node

Run the node visualizer
```
docker run -it -d -p 3000:3000 -e HOST=<pub-ip> -e PORT=3000 -v /var/run/docker.sock:/var/run/docker.sock manomarks/visualizer
```

```
ssh ubuntu@<other-nodes>
sudo docker swarm join <leader-priv-ip>:2377
```

Now, from the swarm leader:

```
docker node ls
```

#### Play with Docker Service concept

From the master node:

```
docker service create --replicas 1 --name helloworld alpine ping docker.com
```

```
docker service ls
docker service inspect --pretty helloworld
docker service scale helloworld=5
```

To see which nodes are running the tasks:
```
docker service tasks helloworld
```

To delete this service
```
docker service rm helloworld
```


See also:

* `docker service create --mode=global`: services required on every node
* `docker service create --constraint com.example.storage=ssd`: assumes `docker daemon --label com.example.storage=ssd`
* [Bring node down for maintenance](https://github.com/mikegcoleman/labs/tree/master/dockercon-us/docker-orchestration#step-34---bring-a-node-down-for-maintenance)

#### Create Distributed Application Bundle (DAB)

[See docker/experimental/dab](https://docker.com/dab)

```
docker-compose bundle
```
*Note*: the .env file is stored within the bundle (this may expose certain secrets)

```
docker deploy ecssample
```

[See DockerCon Demo](https://youtu.be/vE1iDPx6-Ok?t=6405)
```
docker service update -p
```

Ideally we'd use Instance Roles in our Terraform plan for the EC2 Instances created.

## Amazon ECS Introduction

Amazon EC2 Container Service (ECS) is a container management service which allows you to run containers
on a managed cluster of Amazon EC2 Instances. This eliminates the need of installing, operating and
scaling your own cluster management infrastructure.

Amazon ECS ties in with other Amazon Services such as IAM, Elastic Load Balancer, ...

To set up a sample cluster we will  use the [Amazon ECS CLI](https://github.com/aws/amazon-ecs-cli) Tool, see the instructions below.
To set up a cluster follow the [Setting Up](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/get-set-up-for-amazon-ecs.html) instructions on AWS

### Components

- Cluster: EC2 Instances grouped together to run containers.
  Each instance runs the [open source ecs-agent](https://github.com/aws/amazon-ecs-agent) which starts containers on behalf of Amazon ECS.
- Task Definitions: Task Definitions specify the configuration of a group of containers which should be ran on the same host. The concept is similar to the concept of Container Pods.
- Schedulers: The Schedulers used by Amazon ECS define how containers are ran

## Deploying to Amazon

We will be deploying our application containers to an Amazon ECS cluster to easily scale and manage the running instances using containers in a way identical to the stack used for local development. However, for the services (Redis, MongoDB) our application relies on, we will take a slightly different approach.

For Redis we will take advantage of the managed service provided through [Amazon ElastiCache](https://aws.amazon.com/elasticache/) and for MongoDB we will deploy a dedicated cluster outside of Amazon ECS.

As an alternative to the self-service approach required for MongoDB, we could use the fully managed [Amazon DynamoDB](http://aws.amazon.com/dynamodb/) NoSQL database service.

To migrate from MongoDB to DynamoDB we will need to change the implementation of the persistance layer of our application.

For development purposes we may follow [these AWS guidelines](https://aws.amazon.com/blogs/aws/dynamodb-local-for-desktop-development/) to build a DynamoDB Docker Image, as a reference see [this Dockerfile](https://hub.docker.com/r/deangiberson/aws-dynamodb-local/~/dockerfile/).

Once we have an up to date docker DynamoDB image we will also need to make few changes to our `docker-compose.yaml` file to stand up the new stack.

### Deploying the NoSQL Document store

To keep this sample simple, we will use the [MongoDB on AWS Quick Start Guide](http://docs.aws.amazon.com/quickstart/latest/mongodb/overview.html) and create the stack using [Cloudformation from the AWS CLI](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-cli-creating-stack.html). The configurations used are not recommended for a production deploy but are acceptable for demo purposes.

Verify at least 1 key-pair exists to work with EC2 instances:

```
aws ec2 describe-key-pairs
```

If you do not have any key-pair, follow [These instructions](http://docs.aws.amazon.com/cli/latest/reference/ec2/import-key-pair.html) to import a new key pair into EC2.

**Note**: The Quickstart template creates a NAT Instance in a DMZ subnet while keeping the MongoDB Instances in a private subnet. As of December 2015, AWS offers a [Managed EC2 NAT gateway](https://aws.amazon.com/blogs/aws/new-managed-nat-network-address-translation-gateway-for-aws/) and it is no longer required to manage a NAT instance by yourself.

Create a Parameter file for the MongoDb Cloudformation Template:

**Note**: Don't forget to update the name of the key pair
```
cat <<EOF > mongo-stack-parameters.json
[
  {
    "ParameterKey": "AvailabilityZone0",
    "ParameterValue":"ap-southeast-1a"
  },
  {
    "ParameterKey": "AvailabilityZone1",
    "ParameterValue":"ap-southeast-1b"
  },
  {
    "ParameterKey": "AvailabilityZone2",
    "ParameterValue":"ap-southeast-1b"
  },
  {
    "ParameterKey": "ClusterReplicaSetCount",
    "ParameterValue":"1"
  },
  {
    "ParameterKey": "ClusterShardCount",
    "ParameterValue":"0"
  },
  {
    "ParameterKey": "KeyName",
    "ParameterValue":"your-key"
  },
  {
    "ParameterKey": "MongoDBVersion",
    "ParameterValue":"3.0"
  },
  {
    "ParameterKey": "NodeInstanceType",
    "ParameterValue":"m3.medium"
  }
]
EOF
```

Create the MongoDB stack in a new VPC using the QuickStart Template:
```
aws cloudformation create-stack \
  --stack-name "mongo-quickstart" \
  --capabilities CAPABILITY_IAM \
  --template-url https://s3.amazonaws.com/quickstart-reference/mongodb/latest/templates/MongoDB-VPC.template \
  --parameters file://mongo-stack-parameters.json
```

To review the status of each Stack resource using the CLI:
```
aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq '.StackResourceSummaries[] | {Type: .ResourceType, Status: .ResourceStatus}'
```

To Review the status of the overall stack creation process using the CLI:
```
aws cloudformation describe-stacks --stack-name "mongo-quickstart" | jq -r '.Stacks[0].StackStatus'
```

Get the Private IP of the MongoDB Instance (this command may differ for you if you changed the parameters above):

```
MONGO_INSTANCE=`aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "PrimaryReplicaNode0NodeInstanceGP2") | .PhysicalResourceId '`
MONGO_INSTANCE_IP=`aws ec2 describe-instances --instance-ids $MONGO_INSTANCE | jq -r '.Reservations[].Instances[].PrivateIpAddress'`

```

Get the Public IP of the NAT Instance (we may use this as a jump host while troubleshooting):
```
NAT_INSTANCE=`aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "NATInstance") | .PhysicalResourceId '`
NAT_INSTANCE_IP=`aws ec2 describe-instances --instance-ids $NAT_INSTANCE | jq -r '.Reservations[].Instances[].PublicIpAddress'`
ssh ec2-user@$NAT_INSTANCE_IP
```

### Creating the Amazon ElastiCache Service

Get the VPC and Private Subnet of the Mongo Cluster:
```
VPC=`aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "VPC") | .PhysicalResourceId '`

PRIVATE_SUBNET=`aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "PrimaryNodeSubnet") | .PhysicalResourceId '`
```

Create the ElastiCache subnet into the existing private subnet:
```
aws elasticache create-cache-subnet-group --cache-subnet-group-name "ecs-sample-cache" --cache-subnet-group-description "ecs sample" --subnet-ids $PRIVATE_SUBNET
```

Get GroupId of Default Security Group:
```
DEFAULT_SG=`aws ec2 describe-security-groups --filters Name=vpc-id,Values=$VPC | jq '.SecurityGroups[] | select(.GroupName == "default") | .GroupId'`
```

```
aws elasticache create-cache-cluster \
  --engine redis \
  --cache-node-type cache.t2.micro \
  --num-cache-nodes 1 \
  --cache-subnet-group-name ecs-sample-cache \
  --cache-group-ids $DEFAULT_SG
  --cache-cluster-id ecs-sample
```

Finding endpoints (returns endpoint for first Node in Cluster)
```
aws elasticache describe-cache-clusters \
    --cache-cluster-id ecs-sample \
    --show-cache-node-info | jq -r .CacheClusters[].CacheNodes[0].Endpoint.Address
```

### Setting up the ECS Cluster in the existing DMZ Subnet

Amazon ECS clusters are fully customizable through CloudFormation Templates, refer to the [ECS CloudFormation Snippet](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/quickref-ecs.html) in the Amazon Docs as a reference for creating a production ready cluster.

For this sample, however, we will use the `ecs-cli` tool as a means to quickly provision a Sample ECS cluster into the DMZ subnet of our Mongo stack.
Download the latest release of the `ecs-cli` tool with the following commands:

```console
curl -Lo ecs-cli https://s3.amazonaws.com/amazon-ecs-cli/ecs-cli-darwin-amd64-latest
chmod +x ./ecs-cli
ln -s $PWD/ecs-cli /usr/local/bin/ecs-cli
```

Although the `ecs-cli` tool reads AWS credentials from the same `~/.aws/credentials` file as the `aws cli` tool (which we used in the section setting up S3), some additional configuration is required (cluster name and region need to be defined) at the time of writing:

```
ecs-cli configure --cluster "ecs-sample" -r ap-southeast-1
```

Get the VPC and DMZ Subnet of the Mongo Cluster:
```
VPC=`aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "VPC") | .PhysicalResourceId '`

DMZ_SUBNET=`aws cloudformation list-stack-resources --stack-name "mongo-quickstart" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "DMZSubnet") | .PhysicalResourceId '`
```

**NOTE**: Ideally 2 subnets across 2 availability zones are used instead.

Finally, create a sample `ecs-cluster` in the DMZ Subnet of the Mongo Cluster with the following commands:
```
ecs-cli up --keypair <key-pair-name> --capability-iam \
  --size 2 \
  --vpc "$VPC"
  --subnets "$DMZ_SUBNET" \
  --image-id "ami-cf03d2ac" \
  --instance-type t2.micro
```

Where the following parameters were used:

- `--capability-iam`:  Indicates we aknowledge the ecs-cli tool may create IAM resources
- `--size 2` :  Indicates we want 2 EC2 Instances (default = 1)
- `--vpc`:  Takes an existing VPC (the same VPC created earlier for our MongoDB Cluster)
- `--subnets`: Takes a comma separated list of subnets (The Public subnet of our Mongo VPC in this example)
- `--image-id` :  Should specify the up to date Id for the ECS-optimized AMI of the required region, revise based on [updated list here](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-optimized_AMI.html?adbsc=docs_20160617_62622486&adbid=743951143777120256&adbpl=tw&adbpr=66780587).
- `--instance-type` : `t2.micro` equals to 1 vCPU and 1 GB of Memory per instance, which should be sufficient for this demo.

Once creation of the cluster completed, we may test connectivity from any of the EC2 Instances to our MongoDB cluster as follows:

Get the Public IP of such an instance:
```
ECS_ASG=`aws cloudformation list-stack-resources --stack-name "amazon-ecs-cli-setup-ecs-sample" | jq -r '.StackResourceSummaries[] | select(.LogicalResourceId == "EcsInstanceAsg") | .PhysicalResourceId'`

ECS_INSTANCE1=`aws autoscaling describe-auto-scaling-groups --auto-scaling-group-names $ECS_ASG | jq -r .AutoScalingGroups[].Instances[0].InstanceId`

ECS_INSTANCE1_IP=`aws ec2 describe-instances --instance-ids $ECS_INSTANCE1 | jq -r '.Reservations[].Instances[].PublicIpAddress'`
```
**TO BE COMPLETED**
We will need to modify the ECS Security Group to allow SSH access.
```
#TO BE COMPLETED - done through AWS Console for now...
aws ec2 describe-security-groups...
```

Connect to the instance (confirm the identity when prompted):
```
ssh ec2-user@$ECS_INSTANCE1_IP
```

Get a shell into a Mongo container (this will pull the image first)
```
docker run -it --rm mongo:3.2.6 /bin/bash
```

From this shell, test connectivity to the MongoDB instance:
```
mongo <private-ip-of-mongo-instance>:27017
```

### Creating the Task Definition

Task definition under construction, see `ecs-sample-app.json`

Using AWS CLI to deploy full app to ECS - To be completed.

### Expose Application through ELB

**Note**: Enable Websockets / Sticky Sessions on ELB...

## Conclusion

Docker really simplified getting started with a new technology stack with minimum setup of the local machine required.

AWS ECS provides a robust infrastructure to run Docker Containers at scale in production.

Images built are minimal and share layers where possible

Image Sizes:
```
REPOSITORY          TAG                 CREATED             SIZE
ecssample_lb        latest              2 hours ago         182.8 MB
so0k/ecs-sample     dev                 2 hours ago         821.6 MB
so0k/ecs-sample     1.0.0               3 hours ago         14.33 MB
redis               3.2.0-alpine        7 days ago          29.07 MB
mongo               3.2.6               3 weeks ago         313.1 MB
```
