# README

Author: `Prateek Narendra`

## How to run

The `Dockerfile` defines a self-contained Go reference environment. 
Build and run the program using [Docker](https://docs.docker.com/get-started/get-docker/):
```
$ docker build -t challenge .
$ docker run --rm -it challenge --auth=<token>
```
Feel free to modify the `Dockerfile` as you see fit.

If go `1.23` or later is locally installed, run the program directly for convenience:
```
$ go run main.go --auth=<token>
```

## Discard criteria

The discard criteria is based on the lowest freshness score from the room-temperature/open shelf.
