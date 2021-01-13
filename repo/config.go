package repo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/cpacia/openbazaar3.0/version"
	ipfslogging "github.com/ipfs/go-log/writer"
	"github.com/jessevdk/go-flags"
	"github.com/natefinch/lumberjack"
	"github.com/op/go-logging"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultConfigFilename = "openbazaar.conf"
	defaultLogDirname     = "logs"
	defaultLogFilename    = "ob.log"
)

var (
	DefaultHomeDir    = AppDataDir("openbazaar", false)
	defaultConfigFile = filepath.Join(DefaultHomeDir, defaultConfigFilename)

	fileLogFormat   = logging.MustStringFormatter(`%{time:2006-01-02 T15:04:05.000} [%{level}] [%{module}] %{message}`)
	stdoutLogFormat = logging.MustStringFormatter(`%{color:reset}%{color}%{time:15:04:05} [%{level}] [%{module}] %{message}`)
	LogLevelMap     = map[string]logging.Level{
		"debug":    logging.DEBUG,
		"info":     logging.INFO,
		"notice":   logging.NOTICE,
		"warning":  logging.WARNING,
		"error":    logging.ERROR,
		"critical": logging.CRITICAL,
	}

	DefaultMainnetBootstrapAddrs = []string{
		"/ip4/157.245.122.62/tcp/4001/p2p/12D3KooWBnXsKTuuaYFHEZzoTBepD7w2SiwqvqDKHkVEpCZqzPvd",
	}

	DefaultTestnetBootstrapAddrs = []string{
		"",
	}

	DefaultMainnetSNFServers = []string{
		"12D3KooWBnXsKTuuaYFHEZzoTBepD7w2SiwqvqDKHkVEpCZqzPvd",
	}

	DefaultTestnetSNFServers = []string{
		"",
	}
)

// Config defines the configuration options for OpenBazaar.
//
// See loadConfig for details on the configuration load process.
type Config struct {
	ShowVersion            bool     `short:"v" long:"version" description:"Display version information and exit"`
	ConfigFile             string   `short:"C" long:"configfile" description:"Path to configuration file"`
	DataDir                string   `short:"d" long:"datadir" description:"Directory to store data"`
	LogDir                 string   `long:"logdir" description:"Directory to log output."`
	LogLevel               string   `short:"l" long:"loglevel" description:"set the logging level [debug, info, notice, warning, error, critical]" default:"info"`
	BoostrapAddrs          []string `long:"bootstrapaddr" description:"Override the default bootstrap addresses with the provided values"`
	SwarmAddrs             []string `long:"swarmaddr" description:"Override the default swarm addresses with the provided values"`
	GatewayAddr            string   `long:"gatewayaddr" description:"Override the default gateway address with the provided value"`
	StoreAndForwardServers []string `long:"snfserver" description:"A peerID of a store and forward server to use for receiving messages while offline."`
	Testnet                bool     `short:"t" long:"testnet" description:"Use the test network"`
	DisableNATPortMap      bool     `long:"noupnp" description:"Disable use of upnp."`
	IPNSQuorum             uint     `long:"ipnsquorum" description:"The size of the IPNS quorum to use. Smaller is faster but less up-to-date." default:"2"`
	IPNSResolver           string   `long:"ipnsresolver" description:"If a URL is provided here the node will resolve IPNS records by querying this server instead of using the peer-to-peer network."`
	NoIPNSPubsub           bool     `long:"noipnsps" description:"Disable use of IPNS pubsub."`
	ExchangeRateProviders  []string `long:"exchangerateprovider" description:"API URL to use for exchange rates. Must conform to the BitcoinAverage format." default:"https://ticker.openbazaar.org/api"`
	UseSSL                 bool     `long:"ssl" description:"Use SSL on the API"`
	SSLCertFile            string   `long:"sslcertfile" description:"Path to the SSL certificate file"`
	SSLKeyFile             string   `long:"sslkeyfile" description:"Path to the SSL key file"`
	APIUsername            string   `short:"u" long:"apiusername" description:"The username to use with the API authentication"`
	APIPassword            string   `short:"P" long:"apipassword" description:"The password to use with the API authentication"`
	APICookie              string   `long:"apicookie" description:"A cookie to use for authentication in addition or in place of the un/pw. If set the cookie must be put in the request header."`
	APIAllowedIPs          []string `long:"allowedip" description:"Only allow API connections from these IP addresses"`
	APIAllowAllOrigins     bool     `long:"apiallowallorigins" description:"Cors option to allow all origins on the API."`
	APIPublicGateway       bool     `long:"publicgateway" description:"When this option is used only public GET methods will be allowed in the API"`
	Profile                string   `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	CPUProfile             string   `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	IPFSOnly               bool     `long:"ipfsonly" description:"Disable all OpenBazaar functionality except the IPFS networking."`
	EnabledWallets         []string `long:"enabledwallet" description:"Only enable wallets in this list. Available wallets: [BTC, BCH, LTC, ZEC, ETH]"`
	UserAgentComment       string   `long:"uacomment" description:"Comment to add to the user agent."`
	EnableSNFServer        bool     `long:"enablesnfserver" description:"Enable this node to operate as a store-and-forward server."`
	SNFServerPeers         []string `long:"snfpeer" description:"A list of other store-and-forward servers to replicate snf data to. This is only used when the snf server is enabled."`
	Tor                    bool     `long:"tor" description:"Proxy all incoming and outgoing connections over the Tor network exclusively."`
	DualStack              bool     `long:"dualstack" description:"Listen for incoming connections via Tor in addition to via the clearnet. This mode is not private."`
	DHTClientOnly          bool     `long:"dhtclientonly" description:"Disable participating in serving data in the DHT. This should be used if your node is undialable."`
}

// LoadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
// 	1) Start with a default config with sane settings
// 	2) Pre-parse the command line to check for an alternative config file
// 	3) Load configuration file overwriting defaults with any specified options
// 	4) Parse CLI options and overwrite/add any specified options
//
// The above results in OpenBazaar functioning properly without any config settings
// while still allowing the user to override settings with config files and
// command line options.  Command line options always take precedence.
func LoadConfig() (*Config, error) {
	// Default config.
	cfg := Config{
		DataDir:    DefaultHomeDir,
		ConfigFile: defaultConfigFile,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preCfg := cfg
	preParser := flags.NewParser(&cfg, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			return nil, err
		}
	}
	if cfg.DataDir != "" {
		preCfg.ConfigFile = filepath.Join(cfg.DataDir, defaultConfigFilename)
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version.String())
		os.Exit(0)
	}

	// Load additional config from file.
	var configFileError error
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := os.Stat(preCfg.ConfigFile); os.IsNotExist(err) {
		err := createDefaultConfigFile(preCfg.ConfigFile, cfg.Testnet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating a "+
				"default config file: %v\n", err)
		}
	}

	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintf(os.Stderr, "Error parsing config "+
				"file: %v\n", err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
		configFileError = err
	}

	if cfg.Tor && cfg.DualStack {
		return nil, errors.New("tor and dualstack options cannot be used together")
	}

	_, ok := LogLevelMap[strings.ToLower(cfg.LogLevel)]
	if !ok {
		return nil, errors.New("invalid log level")
	}

	cfg.DataDir = cleanAndExpandPath(cfg.DataDir)
	if cfg.LogDir == "" {
		cfg.LogDir = cleanAndExpandPath(path.Join(cfg.DataDir, "logs"))
	}

	// Validate profile port number
	if cfg.Profile != "" {
		profilePort, err := strconv.Atoi(cfg.Profile)
		if err != nil || profilePort < 1024 || profilePort > 65535 {
			str := "%s: The profile port must be between 1024 and 65535"
			return nil, fmt.Errorf(str)
		}
	}

	// Warn about missing config file only after all other configuration is
	// done.  This prevents the warning on help messages and invalid
	// options.  Note this should go directly before the return.
	if configFileError != nil {
		log.Errorf("%v", configFileError)
	}
	return &cfg, nil
}

// SetupLogging sets up logging for this node
func SetupLogging(logDir, logLevel string) {
	backendStdout := logging.NewLogBackend(os.Stdout, "", 0)
	backendStdoutFormatter := logging.NewBackendFormatter(backendStdout, stdoutLogFormat)

	if logDir != "" {
		rotator := &lumberjack.Logger{
			Filename:   path.Join(logDir, defaultLogFilename),
			MaxSize:    10, // Megabytes
			MaxBackups: 3,
			MaxAge:     30, // Days
		}

		backendFile := logging.NewLogBackend(rotator, "", 0)
		backendFileFormatter := logging.NewBackendFormatter(backendFile, fileLogFormat)
		logging.SetBackend(backendStdoutFormatter, backendFileFormatter)

		mirrorWriter := ipfslogging.NewMirrorWriter()
		w2 := &lumberjack.Logger{
			Filename:   path.Join(logDir, "ipfs.log"),
			MaxSize:    10, // Megabytes
			MaxBackups: 3,
			MaxAge:     30, // Days
		}
		mirrorWriter.AddWriter(w2)
	} else {
		logging.SetBackend(backendStdoutFormatter)
	}
	logging.SetLevel(LogLevelMap[strings.ToLower(logLevel)], "")
}

// createDefaultConfig copies the sample-bchd.conf content to the given destination path,
// and populates it with some randomly generated RPC username and password.
func createDefaultConfigFile(destinationPath string, testnet bool) error {
	// Create the destination directory if it does not exists
	err := os.MkdirAll(filepath.Dir(destinationPath), 0700)
	if err != nil {
		return err
	}

	sampleBytes, err := Asset("sample-openbazaar.conf")
	if err != nil {
		return err
	}
	src := bytes.NewReader(sampleBytes)

	dest, err := os.OpenFile(destinationPath,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	// We copy every line from the sample config file to the destination,
	// only replacing the two lines for rpcuser and rpcpass
	reader := bufio.NewReader(src)
	for err != io.EOF {
		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if strings.Contains(line, "bootstrapaddr=") {
			if _, err := dest.WriteString(""); err != nil {
				return err
			}
			if testnet {
				for _, addr := range DefaultTestnetBootstrapAddrs {
					if _, err := dest.WriteString("bootstrapaddr=" + addr + "\n"); err != nil {
						return err
					}
				}
			} else {
				for _, addr := range DefaultMainnetBootstrapAddrs {
					if _, err := dest.WriteString("bootstrapaddr=" + addr + "\n"); err != nil {
						return err
					}
				}
			}
			continue
		}

		if strings.Contains(line, "snfserver=") && !strings.Contains(line, "enablesnfserver=") {
			if _, err := dest.WriteString(""); err != nil {
				return err
			}
			if testnet {
				for _, addr := range DefaultTestnetSNFServers {
					if _, err := dest.WriteString("snfserver=" + addr + "\n"); err != nil {
						return err
					}
				}
			} else {
				for _, addr := range DefaultMainnetSNFServers {
					if _, err := dest.WriteString("snfserver=" + addr + "\n"); err != nil {
						return err
					}
				}
			}
			continue
		}

		if _, err := dest.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(DefaultHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}
