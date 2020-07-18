FritzExporter
===

## Prometheus Exporter for Fritz!Box

We are not affiliated, associated, authorized, endorsed by, or in any way officially connected with AVM. This software is created to make use of the API of the Fritz!Box hardware by AVM to expose information as prometheus readable metrics.

## Usage

```
NAME:
   FritzExporter - FritzExporter

USAGE:
   FritzExporter [global options] command [command options] [arguments...]

VERSION:
   dc38867 (dc38867)

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --fritzbox-url value      URL to connect to [$FRITZ_FRITZBOX_URL]
   --username value          Username to login into Fritz!Box [$FRITZ_USERNAME]
   --log-level value         Only log messages with given severity (default: "info") [$FRITZ_LOG_LEVEL]
   --password value          Password to login into Fritz!Box [$FRITZ_PASSWORD]
   --exporter-address value  Address to bind the metrics server (default: "0.0.0.0:9200") [$FRITZ_EXPORTER_METRICS_ADDRESS]
   --help, -h                Show help (default: false)
   --version, -v             Prints the current version (default: false)
```
