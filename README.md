# s3 tester
```
NAME:
   s3-tester - S3 Tester

USAGE:
   s3-tester [global options] command [command options] [arguments...]

COMMANDS:
   upload, u  Upload a file to the specified S3 bucket
   remove, r  Remove a file from the specified S3 bucket
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose, -v                 debug output (default: false) [$S3_VERBOSE]
   --very-verbose, --vv          trace output (default: false) [$S3_VERY_VERBOSE]
   --endpoint value, -e value    s3 endpoint [$S3_ENDPOINT]
   --port value, -p value        s3 port (default: 0) [$S3_PORT]
   --access-key value, -a value  s3 access key [$S3_ACCESS_KEY]
   --secret-key value, -s value  s3 secret key [$S3_SECRET_KEY]
   --bucket value, -b value      s3 bucket [$S3_BUCKET]
   --insecure                    s3 insecure connection (default: false) [$S3_INSECURE]
   --help, -h                    show help
```