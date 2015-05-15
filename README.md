# heka-s3

Heka output plugin for persisting messages from the data pipeline to AWS S3 buckets.

Sample TOML file (with Kafka as input source):

```
[KafkaInput]
type = "KafkaInput"
topic = "error-logs"
addrs = ["kafka-a-1.test.org:9092"]

[output_s3]
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

[PayloadEncoder]
append_newlines = false
```
`ticker_interval` is in seconds and gzip is used for compression. The plugin will write messages to a buffer which will be uploaded to S3 at every tick. 

In this example, the S3 object will be saved to `/logs/error-logs/2015-05-18/20150518174140.zip`
