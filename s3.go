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
	// MaxBufferSize uint32 `toml:"max_buffer_size"`

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
	tickerChan := or.Ticker()
	// buf := make([]byte, so.config.MaxBufferSize * 1024)
	buffer := bytes.NewBuffer(nil)

	var (
		pack    *PipelinePack
		msg     *message.Message
		// buffer  *bytes.Buffer
		ok      = true
	)

	for ok {
		select {
		case pack, ok = <- inChan:
			if !ok {
				break
			}
			msg = pack.Message
			// or.LogMessage(fmt.Sprintf("writing to buffer"))
			_, err := buffer.Write([]byte(msg.GetPayload()))
			if err != nil {
				or.LogMessage(fmt.Sprintf("warning, unable to write to buffer: %s", err))
				err = nil
				continue
			}
		case <- tickerChan:
			or.LogMessage(fmt.Sprintf("ticker time's up, uploading payload"))
			err := so.Upload(buffer)
			if err != nil {
				or.LogMessage(fmt.Sprintf("warning, unable to upload payload: %s", err))
				err = nil
				continue
			}
			or.LogMessage(fmt.Sprintf("payload uploaded successfully"))
			buffer.Reset()
		}
		pack.Recycle()
	}
	or.LogMessage(fmt.Sprintf("shutting down s3 output runner"))
	return
}

func (so *S3Output) Upload(buffer *bytes.Buffer) (err error) {
	if buffer.Len() == 0 {
		err = errors.New("buffer is empty")
		return
	}
	t := time.Now().Local().Format("20060102150405")
	path := so.config.PathName + "/" + t 
	err = so.bucket.Put(path, buffer.Bytes(), "text/plain", "public-read")
	return
}

func init() {
	RegisterPlugin("S3Output", func() interface{} {
		return new(S3Output)
	})
}
