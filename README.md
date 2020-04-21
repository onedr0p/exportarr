# exportarr

AIO Prometheus Exporter for Sonarr, Radarr or Lidarr(TBD)

[![Docker Pulls](https://img.shields.io/docker/pulls/onedr0p/exportarr)](https://hub.docker.com/r/onedr0p/exportarr)
[![Go Report Card](https://goreportcard.com/badge/github.com/onedr0p/exportarr)](https://goreportcard.com/report/github.com/onedr0p/exportarr)

This is Prometheus Exporter will export metrics gathered from Sonarr or Radarr. This only supports v3 APIs for Sonarr and Radarr. It will not gather metrics from all 3 at once, and instead you need to tell the exporter what metrics you want. Be sure to see the examples below for more information.

## Usage

### Run with Docker Compose

See examples in the [examples/compose](./examples/compose/) directory

### Run with Kubernetes

See examples in the [examples/kubernetes](./examples/kubernetes/) directory, or use the [billimek-charts/sonarr](https://github.com/billimek/billimek-charts/tree/master/charts/sonarr) or [billimek-charts/radarr](https://github.com/billimek/billimek-charts/tree/master/charts/radarr) Helm charts.

### Run from the CLI

```cmd
./exportarr --help
./exportarr sonarr --help
./exportarr radarr --help
```

#### Sonarr

```cmd
./exportarr sonarr \
  --port 9707 \
  --url http://127.0.0.1:8989 \
  --api-key amlmndfb503rfqaa5ln5hj5qkmu3hy18 \
  --enable-episode-quality-metrics
```

Visit http://127.0.0.1:9707/metrics to see Sonarr metrics

#### Radarr

```cmd
./exportarr radarr \
  --port 9708 \
  --url http://127.0.0.1:7878 \
  --api-key amlmndfb503rfqaa5ln5hj5qkmu3hy18
```

Visit http://127.0.0.1:9708/metrics to see Radarr metrics

## Configuration

|       Environment Variable       | CLI Flag                           | Description                                                            | Default   | Required |
|:--------------------------------:|------------------------------------|------------------------------------------------------------------------|-----------|:--------:|
|            **Global**            |                                    |                                                                        |           |          |
|           `LOG_LEVEL`            | `--log-level or -l`                | Set the default Log Level                                              | `INFO`    |    ❌     |
|       **Sonarr or Radarr**       |                                    |                                                                        |           |          |
|              `URL`               | `--url or -u`                      | The full URL to Sonarr or Radarr                                       |           |    ✅     |
|             `APIKEY`             | `--api-key or -a`                  | API Key for Sonarr or Radarr                                           |           |    ✅     |
|   `ENABLE_UNKNOWN_QUEUE_ITEMS`   | `--enable-unknown-queue-items`     | Set to `true` to enable gathering unknown queue items in Queue metrics | `false`   |    ❌     |
|       `BASIC_AUTH_ENABLED`       | `--basic-auth-enabled`             | Set to `true` to enable Basic Auth                                     | `false`   |    ❌     |
|      `BASIC_AUTH_USERNAME`       | `--basic-auth-username`            | Set to your username if enabled Basic Auth                             |           |    ❌     |
|      `BASIC_AUTH_PASSWORD`       | `--basic-auth-password`            | Set to your password if enabled Basic Auth                             |           |    ❌     |
|       `DISABLE_SSL_VERIFY`       | `--disable-ssl-verify`             | Set to `true` to disable SSL verification                              | `false`   |    ❌     |
|              `PORT`              | `--port or -p`                     | The port the exporter will listen on                                   |           |    ✅     |
|           `INTERFACE`            | `--interface or -i`                | The interface IP the exporter will listen on                           | `0.0.0.0` |    ❌     |
|            **Sonarr**            |                                    |                                                                        |           |          |
| `ENABLE_EPISODE_QUALITY_METRICS` | `--enable-episode-quality-metrics` | Set to `true` to enable gathering total episodes by qualities          | `false`   |    ❌     |
