# Nginx Reaper

Nginx Reaper is a tool designed to run as a sidecar container to terminate shutting down Nginx worker processes.

## Table of Contents

- [Description](#description)
- [Configuration](#configuration)
- [Prometheus metrics](#prometheus-metrics)
- [Logs](#logs)
- [Usage](#usage)
- [Development](#development)
- [Documentation](#documentation)

## Description

After configuration reloads, the Nginx master process spawns new worker processes. Existing processes
continue to run until the specified `worker_shutdown_timeout` expires or all clients close connections.
During regular configuration reloads, the system could potentially run out of memory, which results in an OOM.

Thus, Nginx Reaper is responsible for maintaining the number of `nginx: worker process is shutting down`
according to the configuration settings. Running worker processes are sorted by creation time, and the
oldest process is killed until the configured conditions are met.

```
  /nginx-ingress-controller ...
   \_ nginx: master process /usr/bin/nginx -c /etc/nginx/nginx.conf
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process is shutting down
       \_ nginx: worker process
       \_ nginx: worker process
       \_ nginx: worker process
       \_ nginx: worker process
       \_ nginx: cache manager process
```

## Configuration

Nginx Reaper is configured using environment variables:

| Environment variable       | Description                                                                                                            |
|----------------------------|------------------------------------------------------------------------------------------------------------------------|
| `LOG_LEVEL`                | Set the log level (default: `"INFO"`).                                                                                 |                                                                                    
| `REAPER_INTERVAL`          | Interval at which the Reaper terminates shutting down Nginx worker processes (default: `"30s"`).                       | 
| `MAX_SHUTDOWN_WORKERS`     | Maximum number of shutting down Nginx worker processes to keep (default: `255`).                                       | 
| `AVAILABLE_MEMORY_PERCENT` | Minimum percentage of available memory below which shutting down Nginx worker processes are terminated (default: `0`). | 
| `SERVER_ADDR`              | Address at which the HTTP server listens (default: `":11254"`).                                                        |
| `SHUTDOWN_INTERVAL`        | Interval at which the Reaper checks whether Nginx master process is still running (default: `"10s"`).                  |
| `SHUTDOWN_TIMEOUT`         | Maximum duration the Reaper waits for the Nginx master process to terminate (default: `"5m"`).                         |

The percentage of available memory needed can be determined as follows. First, calculate the memory usage
of the current set of active workers (for example, 6 x 100Mi = 600Mi). Next, decide the number of reloads
required between reaper intervals (for example, 2 reloads in 30 seconds, 2 x 600Mi = 1.2Gi) - this will be
the approximate amount of memory that should be kept available. Finally, calculate the percentage of the
total memory (for example, 1.2Gi / 12Gi * 100 = 10%) and set this value in the environment variable
`AVAILABLE_MEMORY_PERCENT`.

Additionally, Nginx Reaper supports limited configuration using HTTP requests to the `/config` endpoint:

| HTTP request                  | Description                                         |
|-------------------------------|-----------------------------------------------------|
| `PUT /config?log-level=debug` | Set the log level to `"DEBUG"` (default: `"INFO"`). |

## Prometheus metrics

Nginx Reaper exports the Prometheus metrics at the `/metrics` endpoint.

E.g. `curl http://localhost:11254/metrics`

```
# HELP nginx_workers_running_current Current number of running Nginx workers by status
# TYPE nginx_workers_running_current gauge
nginx_workers_running_current{status="active"} 4
nginx_workers_running_current{status="shutdown"} 8
# HELP nginx_workers_shutdown_total Total number of shutdown Nginx workers by status
# TYPE nginx_workers_shutdown_total counter
nginx_workers_shutdown_total{status="error"} 0
nginx_workers_shutdown_total{status="terminated"} 16
```

## Logs

Nginx Reaper writes logs to standard output at the `INFO` level by default.

The log level can be changed using either the `LOG_LEVEL` environment variable at application startup or the
HTTP request to the `/config` endpoint at runtime.

E.g. `curl -v -X PUT http://localhost:11254/config?log-level=debug`

**Startup log messages**

```
2024/01/04 08:30:34 INFO Server listening on ":11254"
2024/01/04 08:30:34 INFO Scheduled Nginx Reaper with configuration: interval 10s, max workers to keep 5, target available memory 30%
```

**Scheduled run log messages**

```
2024/01/04 08:30:44 INFO Executing Nginx Reaper with configuration: interval 10s, max workers to keep 5, target available memory 30%

2024/01/04 08:30:44 DEBUG Number of nginx workers shutting down 5 within limit 5
2024/01/04 08:30:44 DEBUG Available memory 262144000/524288000 bytes is 50% and within 45% limit
```

**Nginx workers termination log messages**

```
2024/01/04 09:39:59 WARNING Number of nginx workers shutting down 6 exceeds limit 5
2024/01/04 09:39:59 WARNING Terminating nginx worker process {"pid":121,"name":"nginx","cmdline":"nginx: worker process is shutting down",...}

2024/01/04 09:40:00 WARNING Available memory 223260672/524288000 bytes is 42% and less than 45% limit
2024/01/04 09:40:00 WARNING Terminating nginx worker process {"pid":335,"name":"nginx","cmdline":"nginx: worker process is shutting down",...}
```

**Shutdown log messages**

```
2024/01/04 09:49:39 INFO Nginx Reaper terminated
2024/01/04 09:49:39 INFO Nginx master process is still running {"pid":64,"name":"nginx","cmdline":"nginx: master process /usr/bin/nginx -c /etc/nginx/nginx.conf",...}
2024/01/04 09:49:39 INFO Scheduled Nginx Reaper shutdown handler with interval 10s and timeout 5m0s
2024/01/04 09:49:49 INFO Executing Nginx Reaper shutdown handler with interval 10s and timeout 5m0s
2024/01/04 09:49:49 INFO Nginx master process is still running {"pid":64,"name":"nginx","cmdline":"nginx: master process /usr/bin/nginx -c /etc/nginx/nginx.conf",...}
2024/01/04 09:49:59 INFO Executing Nginx Reaper shutdown handler with interval 10s and timeout 4m50s
2024/01/04 09:49:59 INFO Nginx master process is still running {"pid":64,"name":"nginx","cmdline":"nginx: master process /usr/bin/nginx -c /etc/nginx/nginx.conf",...}
```

## Usage

To run Nginx Reaper locally, run one of the following commands:

```shell
./nginx-reaper
```

```shell
docker run -it --rm nginx-reaper:latest
```

**Kubernetes Specs (incomplete)**

The configuration example below addresses two tasks. First, it implements a graceful Nginx shutdown by
disabling the health check before the actual shutdown occurs. Second, it adds a Nginx Reaper sidecar
container to prevent OOMs by terminating the shutting down Nginx worker processes.

Define `/healthz` endpoint in the Ingress-Nginx Controller ConfigMap to disable health checks when the
maintenance file is created. This way, the Nginx returns a 503 Service Unavailable response to the health
check requests before the actual shutdown occurs.

```yaml
kind: ConfigMap
data:
  server-snippet: |
    location = /healthz {
      if (-f /etc/nginx/maintenance) {
        return 503;
      }
      return 200;
    }
```

Modify the Ingress-Nginx Controller Pod spec to include the `terminationGracePeriodSeconds`, `args`,
`resources.limits.memory`, `readinessProbe`, and `lifecycle.preStop` parameters.

```yaml
kind: Pod
spec:
  terminationGracePeriodSeconds: 300
  containers:
    - name: controller
      args:
        - /nginx-ingress-controller
        - '--shutdown-grace-period=60'
      resources:
        limits:
          memory: 1G
      readinessProbe:
        httpGet:
          path: /healthz
          port: 80
          scheme: HTTP
      lifecycle:
        preStop:
          exec:
            command:
              - touch
              - /etc/nginx/maintenance
```

Note the `--shutdown-grace-period=60` argument of the Ingress-Nginx Controller to allow at least
60 seconds delay between the maintenance file creation and the actual termination of the Nginx process.

To run Nginx Reaper as a sidecar container, add the following to Helm values:

```yaml
controller:
  shareProcessNamespace: true
  extraContainers:
    - name: nginx-reaper
      image: nginx-reaper:latest
      env:
        - name: AVAILABLE_MEMORY_PERCENT
          value: '20'
      volumeMounts:
        - name: cgroup
          readOnly: true
          mountPath: /sys/fs/cgroup
      securityContext:
        privileged: true
        runAsUser: 101
        runAsGroup: 82
        runAsNonRoot: true
        readOnlyRootFilesystem: true
  extraVolumes:
    - name: cgroup
      hostPath:
        path: /sys/fs/cgroup
        type: Directory
```

Note the `shareProcessNamespace`, `volumeMounts`, `securityContext`, and `extraVolumes`. The combination of
these settings allows the container to operate with privileges for reading processes and cgroups information,
while still forcing restrictions for improved security. The container runs as a non-root user in different
IPC, mount, network, PID namespaces from the host, and the root file system is set to read-only, making it
more secure than a typical unprivileged process on the node.

## Development

GoLand or Visual Studio Code are the recommended IDEs for Go development, but if you prefer using the command line,
here are the instructions.

**Prerequisites**

Nginx Reaper requires [Golang 1.24](https://go.dev/dl/) or later.

**Clone the repository**

```shell
git clone git@github.com:acudovs/nginx-reaper.git
cd nginx-reaper
```

**Run the Go format, static analyzer, tests with coverage, and finally build the application**

Notice the `./...` argument, which is a special syntax used in various Go tools to operate on all packages
within the current directory and its subdirectories.

```shell
go fmt ./...

go vet ./...

go test -v -coverprofile=cover.out ./...
go tool cover -func=cover.out

go build nginx-reaper/cmd/nginx-reaper
```

**Docker image**

To create a Docker image, execute

```shell
docker build --force-rm --no-cache -t nginx-reaper:latest .
```

**Update vendor modules**

These commands ensure that the project's dependencies are updated, unnecessary dependencies are removed,
and the vendor directory is updated with the latest dependencies.

```shell
go get -u ./...
go mod tidy
go mod vendor
```

## Documentation

[Ingress-Nginx Controller](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/)

[Nginx worker_shutdown_timeout](https://nginx.org/en/docs/ngx_core_module.html#worker_shutdown_timeout)

[Go User Manual](https://go.dev/doc/)

[Go Style Guide](https://google.github.io/styleguide/go/)
