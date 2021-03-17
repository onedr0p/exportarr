# exportarr

AIO Prometheus Exporter for Sonarr, Radarr or Lidarr(TBD)

[![Go Report Card](https://goreportcard.com/badge/github.com/onedr0p/exportarr)](https://goreportcard.com/report/github.com/onedr0p/exportarr)

This is Prometheus Exporter will export metrics gathered from Sonarr or Radarr. This only supports v3 APIs for Sonarr and Radarr. It will not gather metrics from all 3 at once, and instead you need to tell the exporter what metrics you want. Be sure to see the examples below for more information.

## Usage

### Run with Docker Compose

See examples in the [examples/compose](./examples/compose/) directory

### Run with Kubernetes

See examples in the [examples/kubernetes](./examples/kubernetes/) directory.

### Run with Docker CLI

#### Sonarr
```bash
docker run --name exportarr_sonarr \
  -e port=9707 \
  -e URL="http://192.168.1.1:8989" \
  -e APIKEY="amlmndfb503rfqaa5ln5hj5qkmu3hy18" \
  -e ENABLE_EPISODE_QUALITY_METRICS="false" \
  --restart unless-stopped \
  -p 9707:9707 \
  -d ghcr.io/onedr0p/exportarr:latest exportarr sonarr
```

Visit http://127.0.0.1:9707/metrics to see Sonarr metrics

#### Radarr

```bash
docker run --name exportarr_radarr \
  -e port=9708 \
  -e URL="http://192.168.1.1:7878" \
  -e APIKEY="zmlmndfb503rfqaa5ln5hj5qkmu3hy19" \
  --restart unless-stopped \
  -p 9708:9708 \
  -d ghcr.io/onedr0p/exportarr:latest exportarr radarr
```

Visit http://127.0.0.1:9708/metrics to see Radarr metrics

#### Lidarr

```bash
docker run --name exportarr_lidarr \
  -e port=9709 \
  -e URL="http://192.168.1.1:8686" \
  -e APIKEY="zmlmndfb503rfqaa5ln5hj5qkmu3hy19" \
  --restart unless-stopped \
  -p 9709:9709 \
  -d ghcr.io/onedr0p/exportarr:latest exportarr lidarr
```

Visit http://127.0.0.1:9709/metrics to see Lidarr metrics

### Run from the CLI

```sh
./exportarr --help
```

#### Sonarr

```sh
./exportarr sonarr --help

./exportarr sonarr \
  --port 9707 \
  --url http://127.0.0.1:8989 \
  --api-key amlmndfb503rfqaa5ln5hj5qkmu3hy18 \
  --enable-episode-quality-metrics
```

Visit http://127.0.0.1:9707/metrics to see Sonarr metrics

#### Radarr

```sh
./exportarr radarr --help

./exportarr radarr \
  --port 9708 \
  --url http://127.0.0.1:7878 \
  --api-key amlmndfb503rfqaa5ln5hj5qkmu3hy18
```

Visit http://127.0.0.1:9708/metrics to see Radarr metrics

#### Lidarr

```sh
./exportarr lidarr --help

./exportarr lidarr \
  --port 9709 \
  --url http://127.0.0.1:8686 \
  --api-key amlmndfb503rfqaa5ln5hj5qkmu3hy18
```

Visit http://127.0.0.1:9709/metrics to see Radarr metrics

## Configuration

|       Environment Variable        | CLI Flag                            | Description                                                            | Default   | Required |
|:---------------------------------:| ----------------------------------- | ---------------------------------------------------------------------- | --------- |:--------:|
|            **Global**             |                                     |                                                                        |           |          |
|            `LOG_LEVEL`            | `--log-level or -l`                 | Set the default Log Level                                              | `INFO`    |    ❌    |
|       **Sonarr or Radarr**        |                                     |                                                                        |           |          |
|               `URL`               | `--url or -u`                       | The full URL to Sonarr, Radarr, or Lidarr                             
 |           |    ✅    |
|             `APIKEY`              | `--api-key or -a`                   | API Key for Sonarr, Radarr or Lidarr                               |           |    ✅    |
|   `ENABLE_UNKNOWN_QUEUE_ITEMS`    | `--enable-unknown-queue-items`      | Set to `true` to enable gathering unknown queue items in Queue metrics | `false`   |    ❌    |
|       `BASIC_AUTH_ENABLED`        | `--basic-auth-enabled`              | Set to `true` to enable Basic Auth                                     | `false`   |    ❌    |
|       `BASIC_AUTH_USERNAME`       | `--basic-auth-username`             | Set to your username if enabled Basic Auth                             |           |    ❌    |
|       `BASIC_AUTH_PASSWORD`       | `--basic-auth-password`             | Set to your password if enabled Basic Auth                             |           |    ❌    |
|       `DISABLE_SSL_VERIFY`        | `--disable-ssl-verify`              | Set to `true` to disable SSL verification                              | `false`   |    ❌    |
|              `PORT`               | `--port or -p`                      | The port the exporter will listen on                                   |           |    ✅    |
|            `INTERFACE`            | `--interface or -i`                 | The interface IP the exporter will listen on                           | `0.0.0.0` |    ❌    |
|            **Sonarr**             |                                     |                                                                        |           |          |
| `ENABLE_EPISODE_QUALITY_METRICS`  | `--enable-episode-quality-metrics`  | Set to `true` to enable gathering total episodes by qualities (slow)   | `false`   |    ❌    |
|            **Lidarr**             |                                     |                                                                        |           |          |
|   `ENABLE_SONG_QUALITY_METRICS`   | `--enable-song-quality-metrics`     | Set to `true` to enable gathering total songs by quality (slow)        | `false`   |    ❌    |
| `ENABLE_MONITORED_ALBUMS_METRICS` | `--enable-monitored-albums-metrics` | Set to `true` to enable gathering monitored albums (slow)              | `false`   |    ❌    |
