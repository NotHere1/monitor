# monitor
apache log monitoring

This code try to naively replicate the behavior of the open source apache access log monitor tool [Apachetop](https://linux.die.net/man/1/apachetop).

It was built using Go, which is known for it's easy concurrency primitive and simple binary executable. Those properties make it ideal for designinig application that requires high throughput processing and compact memory footprint.

[Disclaimer]
this is build for training purposes only. not suited for production use.

---

#### required Golang

[Linux Installation](https://github.com/golang/go/wiki/Ubuntu)

---

```sh
# [tested on]
# go version go1.12.1 linux/amd64
#

# create test dir
mkdir -p ~/gotestdir

# init GOPATH
cd ~/gotestdir
export GOPATH=$PWD

# assuming go binary is installed
go get github.com/NotHere1/monitor

# sample usage
cat apache.log | ./monitor -threshold 500 -window 50

# or cli flag with default
./bin/monitor -file apache.log
```

---

_[Scope for Improvement]_

The main idea I chose Golang for this project is to maximize throughput by minizing blocks.

The throughput can be improved by removing unnecessary serialization of the line object during the input scanning phase. Instead, I can just parse the line and take the cols that I needed only.

The throughput can also be further be improved by using workers for the `aggregateLogs` portion of the code to have multiple workers simultaneous collecting read in lines, which once either passed 10 seconds mark or timeout, pass accumulated data to the `summarizeAggregatedLogs` coroutine.

The hotspot on `aggregateLogs` can also be alleviated some by using buffers on the receiving channel.

Those are some of the main ideas that I came out with. There might be more optimizations that I missed or do not know about. I have only been using Golang for < 6 months. So there is still a lot to learn.