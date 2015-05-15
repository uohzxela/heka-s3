# heka-s3

Heka output plugin for persisting messages from the data pipeline to AWS S3 buckets.

Sample TOML file (with Kafka as input source):

```
[KafkaInput]
type = "KafkaInput"
topic = "asdf"
addrs = ["kafka-staging-a-1.bezurk.org:9092"]

[output_s3]
type = "S3Output"
message_matcher = "TRUE"
secret_key = "secret_key"
access_key = "access_key"
bucket = "logs"
prefix = "/error-logs"
region = "ap-southeast-1"
ticker_interval = 360
encoder = "PayloadEncoder"
compression = true

[PayloadEncoder]
append_newlines = false
```
`ticker_interval` is in seconds and gzip is used for compression.
