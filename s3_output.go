package s3

import (
	"fmt"
	"errors"
	"bytes"
	"time"
	"compress/gzip"
	"github.com/mozilla-services/heka/message"
	. "github.com/mozilla-services/heka/pipeline"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

type S3OutputConfig struct {
	SecretKey string `toml:"secret_key"`
	AccessKey string `toml:"access_key"`
	Region string `toml:"region"`
	Bucket string `toml:"bucket"`
	Prefix string `toml:"prefix"`
	TickerInterval uint `toml:"ticker_interval"`
	Compression bool `toml:"compression"`
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
	so.bucket = so.client.Bucket(so.config.Bucket)
	return
}

func (so *S3Output) Run(or OutputRunner, h PluginHelper) (err error) {
	inChan := or.InChan()
	tickerChan := or.Ticker()
	buffer := bytes.NewBuffer(nil)

	var (
		pack    *PipelinePack
		msg     *message.Message
		ok      = true
	)

	for ok {
		select {
		case pack, ok = <- inChan:
			if !ok {
				break
			}
			msg = pack.Message
			_, err := buffer.Write([]byte(msg.GetPayload()))
			if err != nil {
				or.LogMessage(fmt.Sprintf("Warning, unable to write to buffer: %s", err))
				err = nil
				continue
			}
			pack.Recycle()
		case <- tickerChan:
			or.LogMessage(fmt.Sprintf("Ticker fired, uploading payload."))
			err := so.Upload(buffer)
			if err != nil {
				or.LogMessage(fmt.Sprintf("Warning, unable to upload payload: %s", err))
				err = nil
				continue
			}
			or.LogMessage(fmt.Sprintf("Payload uploaded successfully."))
			buffer.Reset()
		}
	}

	or.LogMessage(fmt.Sprintf("Shutting down S3 output runner."))
	return
}

func (so *S3Output) Upload(buffer *bytes.Buffer) (err error) {
	if buffer.Len() == 0 {
		err = errors.New("Buffer is empty.")
		return
	}

	currentTime := time.Now().Local().Format("20060102150405")
	currentDate := time.Now().Local().Format("2006-01-02 15:00:00 +0800")[0:10]
	
	if so.config.Compression { 	
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		writer.Write(buffer.Bytes())
		writer.Close()

		path := so.config.Prefix + "/" + currentDate + "/" + currentTime + ".gz"
		err = so.bucket.Put(path, buf.Bytes(), "multipart/x-gzip", "public-read")
	} else {
		path := so.config.Prefix + "/" + currentDate + "/" + currentTime 
		err = so.bucket.Put(path, buffer.Bytes(), "text/plain", "public-read")
	}

	return
}

func init() {
	RegisterPlugin("S3Output", func() interface{} {
		return new(S3Output)
	})
}
