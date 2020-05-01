# Steps to Deploy DeathStar using Go Script

This readme is meant to provide insight into setting up and running the media-microservices [DeathStarBench](https://github.com/delimitrou/DeathStarBench/tree/master/mediaMicroservices) on a Cloudlab instance with 3 nodes (1 master, 2 worker nodes)

## Setup Steps:
1. Install Docker and Kubernetes and Go
2. Instantiate a K8s cluster
3. Install python3-pip, asyncio, and aiohttp
4. Install k8s client go libraries:
    ```
        go get k8s.io/api
        go get k8s.io/apimachinery
        go get k8s.io/client-go
    ```

## Deploy Application without Go Script
1. Change lines 55, 59, 63, and 67 in `<path-to-repo>/DeathStarBench/mediaMicroservices/k8s-yaml/nginx-web-server.yaml` which refers to the the installation directory location of DeathStarBench to the appropriate location
2. On Master note, create namespace `media-microsvc` via: `kubectl create namespace media-microsvc`
3. Deploy all pods via: `kubectl apply -f <path-of-repo>/mediaMicroservices/k8s-yaml/`
4. Wait until `kubectl -n media-microsvc get pod` displays all Pods as running

## Deploy Application with Go Script
1. Change lines 55, 59, 63, and 67 in `<path-to-repo>/DeathStarBench/mediaMicroservices/k8s-yaml/nginx-web-server.yaml` which refers to the the installation directory location of DeathStarBench to the appropriate location
2. On Master note, create namespace `media-microsvc` via: `kubectl create namespace media-microsvc`
3.
4. Wait until `kubectl -n media-microsvc get pod` displays all Pods as running


## Register users and movie information
1. Use `kubectl -n media-microsvc get svc nginx-web-server` to get its cluster-ip.
2. Paste the cluster ip at `<path-of-repo>/mediaMicroservices/scripts/write_movie_info.py:99` and `<path-of-repo>/mediaMicroservices/scripts/register_users.sh:5`
3.Update `<path-of-repo>/mediaMicroservices/scripts/write_movie_info.py:95 & 97` to point to the installation path of the repo
4. `python3 <path-of-repo>/mediaMicroservices/scripts/write_movie_info.py && <path-of-repo>/mediaMicroservices/scripts/register_users.sh`

## Running HTTP workload generator
A sample workload can be to compose reviews - to do this, do:
1. Paste the cluster ip at `<path-of-repo>/mediaMicroservices/wrk2/scripts/media-microservices/compose-review.lua:1032`
2. 	```
	cd <path-of-repo>/mediaMicroservices/wrk2
	./wrk -D exp -t <num-threads> -c <num-conns> -d <duration> -L -s ./scripts/media-microservices/compose-review.lua http://<cluster-ip>:8080/wrk2-api/review/compose -R <reqs-per-sec>
	```
