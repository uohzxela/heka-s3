# heka-s3

Heka output plugin for persisting messages from the data pipeline to AWS S3 buckets.

Sample TOML file (with Kafka as input source):

```
[error-logs-kafka-input]
type = "KafkaInput"
topic = "error-logs"
addrs = ["kafka-a-1.test.org:9092"]

[error-logs-output-s3]
type = "S3Output"
message_matcher = "TRUE"
secret_key = "secret_key"
access_key = "access_key"
bucket = "logs"
prefix = "/error-logs"
region = "ap-southeast-1"
ticker_interval = 3600
encoder = "PayloadEncoder"
compression = true
buffer_path = "/var/log/heka/buffer/s3"
buffer_chunk_limit = 10000 

[PayloadEncoder]
append_newlines = false
```
`ticker_interval` is in seconds and gzip is used for compression. The plugin will write messages to a buffer which will be uploaded to S3 at every tick. `buffer_chunk_limit` is the limit in bytes for the buffer chunk in memory before it is written to disk.

In this example, an S3 object will be saved hourly as `All Buckets/logs/error-logs/2015-05-18/20150518174140.zip` with the filename as the current timestamp within a folder for the current date.
