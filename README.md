# exportarr

AIO Prometheus Exporter for Sonarr, Radarr or Lidarr(TBD)

[![Docker Pulls](https://img.shields.io/docker/pulls/onedr0p/exportarr)](https://hub.docker.com/r/onedr0p/exportarr)
[![Go Report Card](https://goreportcard.com/badge/github.com/onedr0p/exportarr)](https://goreportcard.com/report/github.com/onedr0p/exportarr)

This is Prometheus Exporter will export metrics gathered from Sonarr, Radarr, or Lidarr. It will not gather metrics from all 3 at once, and instead you need to tell the exporter what metrics you want. Be sure to see the examples below for more information.

## Usage

### Run with Docker Compose

See examples in the [examples/compose](./examples/compose/) directory

### Run with Kubernetes

See examples in the [examples/kubernetes](./examples/kubernetes/) directory

### Run from the CLI

```cmd
./exportarr --help
./exportarr sonarr --help
./exportarr radarr --help
./exportarr lidarr --help
```

#### Sonarr

```cmd
./exportarr --listen-port 9707 \
   sonarr \
   --url http://127.0.0.1:8989 \
   --apikey amlmndfb503rfqaa5ln5hj5qkmu3hy18
   --disable-episode-quality-metrics
```

Visit http://127.0.0.1:9707/metrics to see Sonarr metrics

#### Radarr

```cmd
./exportarr --listen-port 9708 \
   radarr \
   --url http://127.0.0.1:7878 \
   --apikey amlmndfb503rfqaa5ln5hj5qkmu3hy18
```

Visit http://127.0.0.1:9708/metrics to see Radarr metrics

#### Lidarr

```cmd
./exportarr --listen-port 9709 \
   radarr \
   --url http://127.0.0.1:8686 \
   --apikey amlmndfb503rfqaa5ln5hj5qkmu3hy18
```

Visit http://127.0.0.1:9709/metrics to see Lidarr metrics

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

### Sonarr specific

|Name                                    |Description                                                  |Default                |
|----------------------------------------|-------------------------------------------------------------|-----------------------|
|`SONARR_URL`                            |Your Sonarr URL                                              |`http://127.0.0.1:8989`|
|`SONARR_APIKEY`                         |Your Sonarr API Key                                          |                       |
|`SONARR_DISABLE_EPISODE_QUALITY_METRICS`|Disable getting Episode qualities                            |`true`                 |

### Radarr specific

|Name                        |Description                                                  |Default                |
|----------------------------|-------------------------------------------------------------|-----------------------|
|`RADARR_URL`                |Your Radarr URL                                              |`http://127.0.0.1:7878`|
|`RADARR_APIKEY`             |Your Radarr API Key                                          |                       |

### Usage With Lidarr

TBD
