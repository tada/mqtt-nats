// +build !citest

// Copyright 2012-2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

// NOTE: Can test with demo servers.
// nats-pub-repeat -s demo.nats.io <subject> <msg>
// nats-pub-repeat -s demo.nats.io:4443 <subject> <msg> (TLS version)

func showUsageAndExit(exitcode int) {
	flag.PrintDefaults()
	os.Exit(exitcode)
}

func main() {
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	var userCreds = flag.String("creds", "", "User Credentials File")
	var repeat = flag.Int("repeat", 1, "Repeat count")
	var askReply = flag.Int("askreply", 0, "seconds to wait for reply. 0 means don't wait")
	var showHelp = flag.Bool("h", false, "Show help message")
	var rootCA = flag.String("cacert", "", "TLS root certificate")
	var clientKey = flag.String("key", "", "TLS key")
	var clientCert = flag.String("cert", "", "TLS certificate")

	log.SetFlags(0)
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	args := flag.Args()
	if len(args) != 2 {
		showUsageAndExit(1)
	}

	// Connect Options.
	opts := []nats.Option{nats.Name("NATS Sample Publisher")}
	if *rootCA != "" {
		opts = append(opts, nats.RootCAs(*rootCA))
	}
	if *clientCert != "" {
		opts = append(opts, nats.ClientCert(*clientCert, *clientKey))
	}

	// Use UserCredentials
	if *userCreds != "" {
		opts = append(opts, nats.UserCredentials(*userCreds))
	}

	// Connect to NATS
	nc, err := nats.Connect(*urls, opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	subj, msg := args[0], args[1]

	buf := bytes.Buffer{}
	for i := 0; i < *repeat; i++ {
		buf.Reset()
		buf.WriteString(msg)
		buf.WriteByte(' ')
		buf.WriteString(strconv.Itoa(i))
		if *askReply != 0 {
			nc.Request(subj, buf.Bytes(), time.Duration(*askReply)*time.Second)
		} else {
			nc.Publish(subj, buf.Bytes())
		}
	}
	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Published [%s] : '%s'\n", subj, msg)
	}
}
