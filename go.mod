module github.com/tada/mqtt-nats

go 1.14

require (
	github.com/golang/protobuf v1.3.5 // indirect
	github.com/nats-io/nats-server/v2 v2.1.4
	github.com/nats-io/nats.go v1.9.1
	github.com/nats-io/nuid v1.0.1
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
)

replace github.com/mattn/goveralls => github.com/thallgren/goveralls v0.0.6-0.20200407151631-d4d4a8d37dfd
