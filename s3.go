package s3

import (
	"fmt"
	"errors"
	"bytes"
	"time"
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
	TickerInterval uint `toml:"ticker_interval"`
	MaxBufferSize uint32 `toml:"max_buffer_size"`

}

type S3Output struct {
	config *S3OutputConfig
	client *s3.S3
	bucket *s3.Bucket
	stopChan chan bool
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
	tickerChan := or.Ticker()
	so.stopChan = make(chan bool)
	buf := make([]byte, so.MaxBufferSize * 1024)
	buffer := bytes.NewBuffer(buf)

	var pack *PipelinePack
	var msg *message.Message

	ok := true

	for ok {
		select {
		case _, ok = <- so.stopChan:
			continue
		case <- tickerChan:
			for pack = range inChan {
				msg = pack.Message
				n, err := buffer.Write([]byte(msg.GetPayload())
				if err != nil {
					err := so.Upload(buffer)
					if err != nil {
						or.LogMessage(fmt.Sprintf("warning, unable to upload payload when buffer is full: %s", err))
						err = nil
						continue
					}
					pack.Recycle()
					break
				}
				pack.Recycle()
			}
			err := so.Upload(buffer)
			if err != nil {
				or.LogMessage(fmt.Sprintf("warning, unable to upload payload after ticker: %s", err))
				err = nil
				continue
			}
		}
	}

	or.LogMessage("shutting down AWS S3 output runner")
	return
}

func (so *S3Output) Stop() {
	close(so.stopChan)
}

func (so *S3Output) Upload(buffer *Buffer) (err error) {
	t := time.Now().Local().Format("20060102150405")
	path := so.config.PathName + "/" + t 
	err = so.bucket.Put(path, buffer.Bytes(), "text/plain", "public-read")
	buffer.Reset()
	return
}

func init() {
	RegisterPlugin("S3Output", func() interface{} {
		return new(S3Output)
	})
}
