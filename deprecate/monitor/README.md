# monitor
apache log monitoring

This code try to naively replicate the behavior of the open source apache access log monitor tool [Apachetop](https://linux.die.net/man/1/apachetop).

It was built using Go, which is known for it's easy concurrency primitive and simple binary executable. Those properties
make it ideal for designinig application that requires high throughput processing and compact memory footprint.

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
tail <path/to/your/apache.log> | ./bin/monitor
```
