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
   3f6a1d3 (3f6a1d3)

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --fritzbox-url value      URL to connect to [$FRITZ_FRITZBOX_URL]
   --username value          Username to login into Fritz!Box [$FRITZ_USERNAME]
   --log-level value         Only log messages with given severity (default: "info") [$FRITZ_LOG_LEVEL]
   --password value          Password to login into Fritz!Box [$FRITZ_PASSWORD]
   --exporter-address value  Address to bind the metrics server (default: "0.0.0.0:9200") [$FRITZ_EXPORTER_METRICS_ADDRESS]
   --fritz-log-path value    Where to write the log from FritzBox, if unset, it won't be queried [$FRITZ_EXPORTER_LOG_PATH]
   --help, -h                Show help (default: false)
   --version, -v             Prints the current version (default: false)
```

## Available Metrics

```
HELP fritzbox_internet_downstream_current Gauge showing latest internet downstream speed
HELP fritzbox_internet_upstream_current Gauge showing latest internet upstream speed
HELP fritzbox_lan_devices_active Gauge showing active state of device
    labels: ip, mac, name, dev_type
HELP fritzbox_lan_devices_online Gauge showing online state of device
HELP fritzbox_lan_devices_speed Gauge showing speed of device
HELP fritzbox_wlan_devices_speed Gauge showing current speed of wifi device
HELP fritzbox_wlan_devices_speed_max Gauge showing maximum speed of wifi device
    labels: ip, mac, name, dev_type, band, standard, direction
HELP fritzbox_wlan_devices_signal Gauge showing signal strength of wifi devices
    labels: ip, mac, name, dev_type, band, standard
```

FritzBox log file written to local disk (see parameter --fritz-log-path)
