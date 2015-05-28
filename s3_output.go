package s3

import (
	"fmt"
	"errors"
	"io"
	"bufio"
	"bytes"
	"time"
	"os"
	"os/exec"
	"strings"
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
	_, err = os.Stat(so.config.BufferPath)
	if os.IsNotExist(err) {
	 	err = os.MkdirAll(so.config.BufferPath, 0666)
		if err != nil { return }
	}

	err = os.Chdir(so.config.BufferPath)
	if err != nil { return }

	_, err = os.Stat(so.bufferFilePath)
	if os.IsNotExist(err) {
		or.LogMessage("Creating buffer file: " +  so.bufferFilePath)
		w, err := os.Create(so.bufferFilePath)
		w.Close()
		if err != nil { return err }
	}
	
	f, err := os.OpenFile(so.bufferFilePath, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil { return }

	_, err = f.Write(buffer.Bytes()) 
	if err != nil { return }

	f.Close()
	buffer.Reset()

	return
}

func (so *S3Output) ReadFromDisk(or OutputRunner) (buffer *bytes.Buffer, err error) {
	if so.config.Compression {
		or.LogMessage("Compressing buffer file...")
		cmd := exec.Command("gzip", so.bufferFilePath)
		err = cmd.Run()
		if err != nil { 
			return nil, err 
		}
		// rename to original filename without .gz extension
		cmd = exec.Command("mv", so.bufferFilePath + ".gz", so.bufferFilePath)
		err = cmd.Run()
		if err != nil { 
			return nil, err 
		}
	}
	
	or.LogMessage("Uploading, reading from buffer file.")
	fi, err := os.Open(so.bufferFilePath)
	if err != nil { return }
	
	r := bufio.NewReader(fi)
	buffer = bytes.NewBuffer(nil)
	
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF { 
			break 
		}
		if n == 0 {
			break
		}
		_, err = buffer.Write(buf[:n])
		if err != nil { 
			break 
		}
	}

	fi.Close()
	return buffer, err
}

func (so *S3Output) Upload(buffer *bytes.Buffer, or OutputRunner) (err error) {
	_, err = os.Stat(so.bufferFilePath)
	if buffer.Len() == 0 && os.IsNotExist(err) {
		err = errors.New("Nothing to upload.")
		return
	}

	err = so.SaveToDisk(buffer, or)
	if err != nil { return }
	
	buffer, err = so.ReadFromDisk(or)
	if err != nil { return }

	currentTime := time.Now().Local().Format("20060102150405")
	currentDate := time.Now().Local().Format("2006-01-02 15:00:00 +0800")[0:10]
	
	ext := ""
	contentType := "text/plain"

	if so.config.Compression {
		ext = ".gz"
		contentType = "multipart/x-gzip"
	}

	path := so.config.Prefix + "/" + currentDate + "/" + currentTime + ext
	err = so.bucket.Put(path, buffer.Bytes(), contentType, "public-read")

	or.LogMessage("Upload finished, removing buffer file on disk.")
	if err == nil {
		err = os.Remove(so.bufferFilePath)
	}

	return
}

func init() {
	RegisterPlugin("S3Output", func() interface{} {
		return new(S3Output)
	})
}
