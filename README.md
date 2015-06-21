# heka-s3

Heka output plugin for persisting messages from the data pipeline to AWS S3 buckets. It buffers logs to disk locally and uploads periodically to S3. It is currently running reliably in production at [Wego](http://www.wego.com).

## Installation

Refer to: http://hekad.readthedocs.org/en/v0.9.2/installing.html#building-hekad-with-external-plugins

Simply add this line in _{heka root}/cmake/plugin_loader.cmake_:

    add_external_plugin(git https://github.com/uohzxela/heka-s3 master)

Then run build.sh as per the documentation.

## Configuration

Sample TOML file (with Kafka as input source):

```
[error-logs-input-kafka]
type = "KafkaInput"
topic = "error-logs"
addrs = ["kafka-a-1.test.org:9092"]

[error-logs-output-s3]
type = "S3Output"
message_matcher = "Logger == 'error-logs-input-kafka'"
secret_key = "SECRET_KEY"
access_key = "ACCESS_KEY"
bucket = "logs"
prefix = "/error-logs"
region = "ap-southeast-1"
ticker_interval = 3600
compression = true
buffer_path = "/var/log/heka/buffer/s3"
buffer_chunk_limit = 1000000
encoder = "PayloadEncoder"

[PayloadEncoder]
append_newlines = false
```

| Attributes        | Type          | Default | Remarks   |
| -------------     |-------------  | -----   | --------- |
| secret_key        | string        |   nil   | needed for S3 authentication |
| access_key        | string        |   nil   | needed for S3 authentication |
| bucket            | string        |   nil   | specifies bucket in S3 |
| prefix            | string        |   nil   | specifies path in bucket |
| region            | string        |   nil   | e.g. "ap-southeast-1"|
| ticker_interval   | int (seconds) |   nil   | specifies buffering time before every upload |
| compression       | boolean       |   true  | only gzip is supported for now |
| buffer_path       | string        |   nil   | defines path to store buffer file locally |
| buffer_chunk_limit| int (bytes)   | 1000000 | defines buffer size limit in memory before flushing to disk|

Logs are saved to S3 as _{bucket}/{prefix}/{current date}/{current time stamp}.gz_, if compression is enabled.

In this example, an S3 object will be saved hourly as `All Buckets/logs/error-logs/2015-05-18/20150518174140.gz`

Regardless of the ticker_interval, buffer on the local disk is automatically uploaded by midnight to the previous day's folder so as to deal with timestamp issue.

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request

## Copyright

Copyright 2015 Alex Jiao Ziheng

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
