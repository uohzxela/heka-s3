package s3

import (
	"fmt"
	"errors"
	"github.com/mozilla-services/heka/message"
	. "github.com/mozilla-services/heka/pipeline"
    "github.com/mitchellh/goamz/aws"
  	"github.com/mitchellh/goamz/s3"
)

type S3OutputConfig struct {
	SecretKey string `toml:"secret_key"`
	AccessKey string `toml:"access_key"`
	Region string `toml:"region"`
	BucketName string `toml:"bucket_name"`
	PathName string `toml:"path_name"`

}

type S3Output struct {
	config *S3OutputConfig
	client *s3.S3
	bucket *s3.Bucket
}

func (so *S3Output) ConfigStruct() interface{} {
	return &S3OutputConfig{}
}

func (so *S3Output) Init(config interface{}) (err error) {
	so.config = config.(*S3OutputConfig)
	auth, err := aws.GetAuth(so.config.AccessKey, so.config.SecretKey)
	if err != nil {
		return
	}
	region, ok := aws.Regions[so.config.Region]
	if !ok {
		err = errors.New("Region of that name not found.")
		return
	}
	so.client = s3.New(auth, region)
  	so.bucket = so.client.Bucket(so.config.BucketName)
	return
}

func (so *S3Output) Run(or OutputRunner, h PluginHelper) (err error) {
	inChan := or.InChan()

	var pack *PipelinePack
	var msg *message.Message

	for pack = range inChan {
		msg = pack.Message
		err = so.bucket.Put(so.config.PathName, []byte(msg.GetPayload()), "text/plain", "public-read")
		if err != nil {
			or.LogMessage(fmt.Sprintf("warning, unable to parse payload: %s", err))
			err = nil
			continue
		}
		pack.Recycle()
	}
	or.LogMessage("shutting down AWS S3 output runner")
	return
}

// func (so *KafkaOutput) CleanupForRestart() {
// 	so.client.Close()
// 	ao.producer.Close()
// 	ao.init()
// }

// init aws client
// func (ao *KafkaOutput) init() (err error) {
// 	cconf := sarama.NewClientConfig()
// 	ao.client, err = sarama.NewClient(ao.config.Id, ao.addrs, cconf)
// 	if err != nil {
// 		return
// 	}
// 	kconf := sarama.NewProducerConfig()
// 	ao.producer, err = sarama.NewProducer(ao.client, kconf)
// 	if err != nil {
// 		return
// 	}
// 	return
// }

func init() {
	RegisterPlugin("S3Output", func() interface{} {
		return new(S3Output)
	})
}
