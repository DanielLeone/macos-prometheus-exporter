set -o errexit -o xtrace

outputPath='./bin/macos_prometheus_exporter'
if [ -b "$1" ]; then
  outputPath=$1
fi

test -f go.mod
test -f ./cmd/macos_prometheus_exporter/main.go
go build -o "${outputPath}" ./cmd/macos_prometheus_exporter/main.go
