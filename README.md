# exportarr

Prometheus Exporter for Sonarr, Radarr and Lidarr(TBD)

[![Docker Pulls](https://img.shields.io/docker/pulls/onedr0p/exportarr)](https://hub.docker.com/r/onedr0p/exportarr)
[![Go Report Card](https://goreportcard.com/badge/github.com/onedr0p/exportarr)](https://goreportcard.com/report/github.com/onedr0p/exportarr)

## Environment Variables

### Global

|Name                        |Description                                                  |Default                |
|----------------------------|-------------------------------------------------------------|-----------------------|
|`BASIC_AUTH_ENABLED`        |Set to `true` to enable Basic Auth                           |`false`                |
|`BASIC_AUTH_USERNAME`       |Set to your username if enabled Basic Auth                   |                       |
|`BASIC_AUTH_PASSWORD`       |Set to your password if enabled Basic Auth                   |                       |
|`DISABLE_SSL_VERIFY`        |Set to `true` to disable SSL verification (use with caution) |`false`                |
|`LISTEN_PORT`               |The port the exporter will listen on                         |`9707`                 |
|`LISTEN_IP`                 |The IP the exporter will listen on                           |`0.0.0.0`              |
|`LOG_LEVEL`                 |Set the default Log Level                                    |`INFO`                 |

## Sonarr specific

|Name                                    |Description                                                  |Default                |
|----------------------------------------|-------------------------------------------------------------|-----------------------|
|`SONARR_URL`                            |Your Sonarr URL                                              |`http://127.0.0.1:8989`|
|`SONARR_APIKEY`                         |Your Sonarr API Key                                          |                       |
|`SONARR_DISABLE_EPISODE_QUALITY_METRICS`|Disable getting Episode qualities                            |`true`                 |

## Radarr specific

|Name                        |Description                                                  |Default                |
|----------------------------|-------------------------------------------------------------|-----------------------|
|`RADARR_URL`                |Your Radarr URL                                              |`http://127.0.0.1:7878`|
|`RADARR_APIKEY`             |Your Radarr API Key                                          |                       |

## Usage With Lidarr

TBD

## Run with Docker Compose

See examples in the [examples/compose](./examples/compose/) directory

## Run with Kubernetes

See examples in the [examples/compose](./examples/kubernetes/) directory

## Run with the command line

```cmd
./exportarr --help

NAME:
   Exportarr - A Prometheus Exporter for Sonarr, Radarr and Lidarr

USAGE:
   exportarr [global options] command [command options] [arguments...]

AUTHOR:
   onedr0p <onedr0p@users.noreply.github.com>

COMMANDS:
   radarr, r  Use the exporter for Radarr
   sonarr, s  Use the exporter for Sonarr
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-level value            Set the default Log Level (default: "INFO") [$LOG_LEVEL]
   --listen-port value          Port the exporter will listen on (default: 9707) [$LISTEN_PORT]
   --listen-ip value            IP the exporter will listen on (default: "0.0.0.0") [$LISTEN_IP]
   --disable-ssl-verify         Disable SSL Verifications (use with caution) (default: false) [$DISABLE_SSL_VERIFY]
   --basic-auth-enabled         Enable Basic Auth (default: false) [$BASIC_AUTH_ENABLED]
   --basic-auth-username value  If Basic Auth is enabled, provide the username (default: "") [$BASIC_AUTH_USERNAME]
   --basic-auth-password value  If Basic Auth is enabled, provide the password (default: "") [$BASIC_AUTH_PASSWORD]
   --help, -h                   show help (default: false)
```

### Sonarr

```cmd
./exportarr --listen-port 9707 \
   sonarr \
   --url http://127.0.0.1:8989 \
   --apikey amlmndfb503rfqaa5ln5hj5qkmu3hy18
   --disable-episode-quality-metrics
```
Visit http://127.0.0.1:9707/metrics to see Sonarr metrics

### Radarr

```cmd
./exportarr --listen-port 9708 \
   radarr \
   --url http://127.0.0.1:7878 \
   --apikey amlmndfb503rfqaa5ln5hj5qkmu3hy18
```

Visit http://127.0.0.1:9708/metrics to see Radarr metrics

### Lidarr

TBD