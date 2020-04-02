package cli

import (
	"flag"
	"io"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/bridge"
	"github.com/tada/mqtt-nats/logger"
)

func Bridge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.SetOutput(stderr)

	var (
		printHelp      bool
		natsClientCert string
		natsClientKey  string
		natsRootCAs    string
		natsCredsFile  string
	)
	opts := &bridge.Options{}
	fs.StringVar(&opts.NATSUrls, "natsurl", nats.DefaultURL, "NATS server URLs separated by comma")
	fs.IntVar(&opts.Port, "port", 0, "MQTT Port to listen on (defaults to 1883 or 8883 with TLS)")
	fs.BoolVar(&printHelp, "h", false, "")
	fs.BoolVar(&printHelp, "help", false, "Print this help")
	fs.IntVar(&opts.RepeatRate, "repeatrate", 5000, "time in milliseconds between each publish of unacknowledged messages")
	// persistence
	fs.StringVar(&opts.StoragePath, "storage", "mqtt-nats.json", "path to json file where server state is persisted")

	fs.BoolVar(&opts.Debug, "D", false, "Enable Debug logging")
	fs.BoolVar(&opts.Debug, "debug", false, "Enable Debug logging")

	// tls
	fs.BoolVar(&opts.TLS, "tls", false, "Enable TLS. If true, the -tlscert and -tlskey options are mandatory")
	fs.StringVar(&opts.TLSCert, "tlscert", "", "Server certificate file")
	fs.StringVar(&opts.TLSKey, "tlskey", "", "Private key for server certificate")

	// options to verify client certificate
	fs.BoolVar(&opts.TLSVerify, "tlsverify", false,
		"Enable verification of client TLS certificate. If true, the -tlscacert option is mandatory")
	fs.StringVar(&opts.TLSCaCert, "tlscacert", "", "Root Certificate for verification of client TLS certificate")

	fs.StringVar(&natsCredsFile, "nats-creds", "", "User Credentials File used when bridge connects to NATS")
	// tls when connecting to the NATS server
	fs.StringVar(&natsClientKey, "nats-key", "", "Public Key used by the bridge when connecting to NATS")
	fs.StringVar(&natsClientCert, "nats-cert", "", "Client Certificate used by the bridge when connecting to NATS")
	fs.StringVar(&natsRootCAs, "nats-cacert", "", "Client Root Certificate used by the bridge when connecting to NATS")

	_ = fs.Parse(args[1:])
	if printHelp {
		fs.SetOutput(stdout)
		fs.PrintDefaults()
		return 0
	}

	if opts.TLS {
		if opts.TLSCert == "" || opts.TLSKey == "" {
			_, _ = io.WriteString(stderr, "both -tlscert and -tlskey must be given when tls is enabled")
			return 2
		}

		if opts.Port == 0 {
			opts.Port = 8883
		}
	} else if opts.Port == 0 {
		opts.Port = 1883
	}

	opts.NATSOpts = []nats.Option{nats.Name("MQTT Bridge")}
	if natsCredsFile != "" {
		opts.NATSOpts = append(opts.NATSOpts, nats.UserCredentials(natsCredsFile))
	}
	if natsClientCert != "" || natsClientKey != "" {
		if natsClientCert == "" || natsClientKey == "" {
			_, _ = io.WriteString(stderr, "both -nats-cert and -nats-key must be given to enable client verification")
			return 2
		}
		opts.NATSOpts = append(opts.NATSOpts, nats.ClientCert(natsClientCert, natsClientKey))
	}
	if natsRootCAs != `` {
		opts.NATSOpts = append(opts.NATSOpts, nats.RootCAs(natsRootCAs))
	}

	lg := logger.New(logger.Debug, stdout, stderr)
	s, err := bridge.New(opts, lg)
	if err == nil {
		err = s.Serve(nil)
	}

	if err != nil {
		lg.Error(err)
		return 1
	}
	return 0
}
