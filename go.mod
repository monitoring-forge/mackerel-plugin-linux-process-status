module github.com/monitoring-forge/mackerel-plugin-linux-process-status

go 1.26.4

require (
	github.com/jessevdk/go-flags v1.6.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/procfs v0.21.1
)

require github.com/monitoring-forge/flagrun v0.0.8

require github.com/mackerelio/checkers v0.2.1 // indirect

require (
	github.com/mackerelio/golib v1.2.2
	github.com/monitoring-forge/saferio v0.0.2
	golang.org/x/sys v0.47.0 // indirect
)
