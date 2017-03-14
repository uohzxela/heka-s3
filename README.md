# Warning

This version was changed to be possible use with the latest heka code base (dev branch)

1. You should include the line `add_external_plugin(git https://github.com/jpereira/heka-s3 master)` before the `include(plugin_loader OPTIONAL)`, the result should be:

```
chessus@vagrant-ubuntu-trusty-64:~/heka$ git status
On branch dev
Your branch is up-to-date with 'origin/dev'.

Changes to be committed:
  (use "git reset HEAD <file>..." to unstage)

	modified:   cmake/externals.cmake

chessus@vagrant-ubuntu-trusty-64:~/heka$ git diff HEAD
diff --git a/cmake/externals.cmake b/cmake/externals.cmake
index 213f6c6..a1eb48b 100644
--- a/cmake/externals.cmake
+++ b/cmake/externals.cmake
@@ -197,6 +197,9 @@ git_clone(https://github.com/gogo/protobuf 7d21ffbc76b992157ec7057b69a1529735fba
 add_custom_command(TARGET protobuf POST_BUILD
 COMMAND ${GO_EXECUTABLE} install github.com/gogo/protobuf/protoc-gen-gogo)
 
+# heka-s3
+add_external_plugin(git https://github.com/jpereira/heka-s3 master)
+
 include(plugin_loader OPTIONAL)
 
 if (PLUGIN_LOADER)
chessus@vagrant-ubuntu-trusty-64:~/heka$ 
```

2. if you're building the heka using the golang >= 1.6, you should declare `export GODEBUG="cgocheck=0"` first of run the `hekad`

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

## License

MIT
