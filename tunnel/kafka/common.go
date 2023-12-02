package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	LOG "github.com/vinllen/log4go"

	"github.com/Shopify/sarama"
	utils "github.com/alibaba/MongoShake/v2/common"
	"github.com/rcrowley/go-metrics"
)

var (
	topicDefault           = "mongoshake"
	topicSplitter          = "@"
	brokersSplitter        = ","
	defaultPartition int32 = 0
)

type Message struct {
	Key       []byte
	Value     []byte
	Offset    int64
	TimeStamp time.Time
}

type Config struct {
	Config *sarama.Config
}

func NewConfig(rootCaFile string) (*Config, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V0_10_0_0
	config.MetricRegistry = metrics.NewRegistry()

	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewManualPartitioner
	config.Producer.MaxMessageBytes = 16*utils.MB + 2*utils.MB // 2MB for the reserve gap

	// ssl
	if rootCaFile != "" {
		sslConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		caCert, err := ioutil.ReadFile(rootCaFile)
		if err != nil {
			LOG.Critical("failed to load the ca cert file[%s]: %s failed: %s", rootCaFile, err.Error())
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		sslConfig.RootCAs = caCertPool
		config.Net.TLS.Config = sslConfig
		config.Net.TLS.Enable = true
	}
	
	user, userExists := os.LookupEnv("KAFKA_USER")
	password, passwordExists:= os.LookupEnv("KAFKA_PASSWORD")
	if userExists && passwordExists {
		LOG.Info("Kafka SASL/SCRAM authentication enabled")
		config.Net.SASL.Enable = true
		config.Net.SASL.User = user
		config.Net.SASL.Password = password
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }

		tlsConfig := tls.Config{}
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tlsConfig
	}

	return &Config{
		Config: config,
	}, nil
}

// parse the address (topic@broker1,broker2,...)
func parse(address string) (string, []string, error) {
	arr := strings.Split(address, topicSplitter)
	l := len(arr)
	if l == 0 || l > 2 {
		return "", nil, fmt.Errorf("address format error")
	}

	topic := topicDefault
	if l == 2 {
		topic = arr[0]
	}

	brokers := strings.Split(arr[l-1], brokersSplitter)
	return topic, brokers, nil
}
