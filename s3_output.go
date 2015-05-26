package s3

import (
	"fmt"
	"errors"
	"bytes"
	"time"
	"os"
	"strings"
	"io/ioutil"
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
	BufferPath string  `toml:"buffer_path"`
	BufferChunkLimit int  `toml:"buffer_chunk_limit"`
	
}

type S3Output struct {
	config *S3OutputConfig
	client *s3.S3
	bucket *s3.Bucket
	bufferFilePath string
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

	prefixList := strings.Split(so.config.Prefix, "/")
	bufferFileName := so.config.Bucket + strings.Join(prefixList, "_")
	so.bufferFilePath = so.config.BufferPath + "/" + bufferFileName
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
			err := so.WriteToBuffer(buffer, msg, or)
			if err != nil {
				or.LogMessage(fmt.Sprintf("Warning, unable to write to buffer: %s", err))
				err = nil
				continue
			}
			pack.Recycle()
		case <- tickerChan:
			or.LogMessage(fmt.Sprintf("Ticker fired, uploading payload."))
			err := so.Upload(buffer, or)
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

func (so *S3Output) WriteToBuffer(buffer *bytes.Buffer, msg *message.Message, or OutputRunner) (err error) {
	_, err = buffer.Write([]byte(msg.GetPayload()))
	if err != nil {
		return
	}
	if buffer.Len() > so.config.BufferChunkLimit {
		err = so.SaveToDisk(buffer, or)
	}
	return
}

func (so *S3Output) SaveToDisk(buffer *bytes.Buffer, or OutputRunner) (err error) {
	var (
		ok bool
		f *os.File
	)

	if ok, err = exists(so.config.BufferPath); err != nil {
		return err
	}

	if !ok {
	 	err = os.MkdirAll(so.config.BufferPath, 0666)
		if err != nil {
			return
		}
	}

	if err = os.Chdir(so.config.BufferPath); err != nil {
		return
	}

	if ok, err = exists(so.bufferFilePath); err != nil {
		return
	}

	if !ok {
		or.LogMessage("Creating buffer file: " +  so.bufferFilePath)
		w, err := os.Create(so.bufferFilePath)
		w.Close()
		if err != nil {
			return err
		}
	}
	
	// or.LogMessage("appending to buffer file")
	if f, err = os.OpenFile(so.bufferFilePath, os.O_APPEND|os.O_WRONLY, 0666); err != nil {
		return
	}
	if _, err = f.Write(buffer.Bytes()); err != nil {
	    return
	}
	f.Close()

	buffer.Reset()
	return
}

func (so *S3Output) ReadFromDisk() (buffer *bytes.Buffer, err error) {
	buf, err := ioutil.ReadFile(so.bufferFilePath)
	buffer = bytes.NewBuffer(buf)

	return buffer, err
}

func (so *S3Output) Upload(buffer *bytes.Buffer, or OutputRunner) (err error) {
	if err := so.SaveToDisk(buffer, or); err != nil {
		return err
	}
	or.LogMessage("Uploading, reading from buffer file.")
	if buffer, err = so.ReadFromDisk(); err != nil {
		return err
	}

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

	or.LogMessage("Upload finished, removing buffer file on disk.")
	if err == nil {
		err = os.Remove(so.bufferFilePath)
	}

	return
}

func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return false, err
}

func init() {
	RegisterPlugin("S3Output", func() interface{} {
		return new(S3Output)
	})
}
