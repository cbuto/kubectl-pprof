# kubectl pprof plugin

A simple kubectl plugin to collect Go pprof profiles from pods that 
expose the ["net/http/pprof"](https://pkg.go.dev/net/http/pprof) endpoints on a local port.

The plugin will port-forward to the specified pod and write the pprof profile to the filesystem
to be analyze with `go tool pprof`.

## Installing

1. Download the binary from the GH release
2. Move the `kubectl-pprof` binary to anywhere in your `$PATH`
3. Run `kubectl pprof -h` to validate the plugin is working

## Example usage

### Collecting a CPU profile

```bash
kubectl pprof <pod> --port 8080 -n <namespace> --profile cpu
```

### Collecting a heap profile for 30 seconds

```bash
kubectl pprof <pod> --port 8080 -n <namespace> --profile heap --seconds 30 --output /tmp/
```

### Pass a profile directly to `go tool pprof` (suppress output with `-q`)

```bash
kubectl pprof <pod> --port 8080 -n <namespace> --profile cpu -q | xargs go tool pprof -http=:8080
```
