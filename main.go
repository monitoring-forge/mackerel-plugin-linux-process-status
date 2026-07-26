package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/mackerelio/golib/pluginutil"
	"github.com/pkg/errors"
	"github.com/prometheus/procfs"
)

var version string
var commit string

const UNKNOWN = 3
const CRITICAL = 2
const WARNING = 1
const OK = 0

type Opt struct {
	Pid       int    `short:"p" long:"pid" description:"PID" required:"true"`
	KeyPrefix string `long:"key-prefix" description:"Metric key prefix" required:"true"`
	Version   bool   `short:"v" long:"version" description:"Show version"`
}

type processStats struct {
	Now    uint64  `json:"now"`
	SysCPU float64 `json:"syscpu"`
	CPU    float64 `json:"cpu"`
}

// cpuJifferAt reads CPU statistics from a given procfs root path.
// This variant is testable with PseudoFS.
func cpuJifferAt(root string) (float64, error) {
	fs, err := procfs.NewFS(root)
	if err != nil {
		return 0, err
	}
	cpu, err := fs.Stat()
	if err != nil {
		return 0, err
	}
	return (cpu.CPUTotal.User + cpu.CPUTotal.Nice + cpu.CPUTotal.System + cpu.CPUTotal.Idle), nil
}

func (opt *Opt) fdsStat(p procfs.Proc, now uint64) (string, error) {
	fds, err := p.FileDescriptorsLen()
	if err != nil {
		return "", errors.Wrap(err, "Could not get fds")
	}

	limit, err := p.Limits()
	if err != nil {
		return "", errors.Wrap(err, "Could not get limits")
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "process-status.fds_%s.count\t%d\t%d\n", opt.KeyPrefix, fds, now)
	fmt.Fprintf(&buf, "process-status.fds_%s.max\t%d\t%d\n", opt.KeyPrefix, limit.OpenFiles, now)
	usage := 0.0
	if limit.OpenFiles > 0 {
		usage = float64(fds) * 100 / float64(limit.OpenFiles)
	}
	fmt.Fprintf(&buf, "process-status.fds_usage_%s.percentage\t%f\t%d\n", opt.KeyPrefix, usage, now)

	return buf.String(), nil
}

func (opt *Opt) memStat(p procfs.Proc, now uint64) (string, error) {
	return memStatAt(p, opt.KeyPrefix, now, "/proc")
}

// memStatAt reads memory statistics from a given procfs root path.
// This variant is testable with PseudoFS.
func memStatAt(p procfs.Proc, keyPrefix string, now uint64, root string) (string, error) {
	pss, err := p.Stat()
	if err != nil {
		return "", errors.Wrap(err, "Could not get process stat")
	}
	used := pss.ResidentMemory()

	fs, err := procfs.NewFS(root)
	if err != nil {
		return "", errors.Wrap(err, "Could not get procfs")
	}
	ms, err := fs.Meminfo()
	if err != nil {
		return "", errors.Wrap(err, "Could not get meminfo")
	}
	// XXX use MemTotal as max memory. not concern cgroup
	memTotal := ms.MemTotal
	if memTotal == nil {
		return "", errors.New("Could not get MemTotal")
	}
	max := *memTotal * 1024

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "process-status.mem_%s.used\t%d\t%d\n", keyPrefix, used, now)
	fmt.Fprintf(&buf, "process-status.mem_%s.max\t%d\t%d\n", keyPrefix, max, now)
	fmt.Fprintf(&buf, "process-status.mem_usage_%s.percentage\t%f\t%d\n", keyPrefix, float64(used)*100/float64(max), now)
	return buf.String(), nil
}

func (opt *Opt) cpuStat(p procfs.Proc, workDir string, now uint64) (string, error) {
	return cpuStatAt(p, opt, now, workDir, "/proc")
}

// cpuStatAt computes CPU statistics from a given procfs root path.
// This variant is testable with PseudoFS.
func cpuStatAt(p procfs.Proc, opt *Opt, now uint64, workDir string, root string) (string, error) {
	pss, err := p.Stat()
	if err != nil {
		return "", errors.Wrap(err, "Could not get process stat")
	}

	c, err := cpuJifferAt(root)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch /proc/stat")
	}

	ps := &processStats{
		Now:    now,
		SysCPU: c,
		CPU:    pss.CPUTime(),
	}

	curUID := os.Geteuid()
	executable, err := os.Executable()
	if err != nil || executable == "" {
		executable = "unknown"
	}
	executable = url.QueryEscape(filepath.Base(executable))
	stateFile := fmt.Sprintf("%d-process-status-v2-%s-%s", curUID, opt.KeyPrefix, executable)

	defer func() {
		if writeErr := writeStats(workDir, stateFile, ps); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to save stats to %s: %v\n", stateFile, writeErr)
		}
	}()

	if !fileExists(workDir, stateFile) {
		fmt.Fprintf(os.Stderr, "Notice: first time execution command\n")
		return "", nil
	}

	prev, err := readStats(workDir, stateFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to load stats")
	}

	if ps.SysCPU-prev.SysCPU == 0 {
		fmt.Fprintf(os.Stderr, "Notice: System CPU counter seems to be unchanged\n")
		return "", nil
	}

	us := (float64(ps.CPU-prev.CPU) / float64(ps.SysCPU-prev.SysCPU)) * 100
	if us < 0 {
		fmt.Fprintf(os.Stderr, "Notice: Process or System CPU counter seems to be reset\n")
		return "", nil
	}

	return fmt.Sprintf("process-status.cpu_%s.percentage\t%f\t%d\n", opt.KeyPrefix, us, now), nil
}

func (opt *Opt) run() error {

	now := uint64(time.Now().Unix())

	proc, err := procfs.NewProc(opt.Pid)
	if err != nil {
		return errors.Wrap(err, "failed to fetch proc")
	}

	msg, err := opt.fdsStat(proc, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Notice: %v\n", err)
	}
	fmt.Print(msg)

	workDir := pluginutil.PluginWorkDir()
	msg, err = opt.cpuStat(proc, workDir, now)
	if err != nil {
		return errors.Wrap(err, "failed to get cpu stat")
	}
	fmt.Print(msg)

	msg, err = opt.memStat(proc, now)
	if err != nil {
		return err
	}
	fmt.Print(msg)

	return nil
}

func main() {
	os.Exit(_main())
}

func _main() int {
	opt := &Opt{}
	psr := flags.NewParser(opt, flags.HelpFlag|flags.PassDoubleDash)
	_, err := psr.Parse()

	if opt.Version {
		if commit == "" {
			commit = "dev"
		}
		fmt.Printf(
			"%s-%s\n%s/%s, %s, %s\n",
			filepath.Base(os.Args[0]),
			version,
			runtime.GOOS,
			runtime.GOARCH,
			runtime.Version(),
			commit)
		return OK
	}
	// flags.PrintErrors is not set, so we need to display help and errors manually
	if err != nil && flags.WroteHelp(err) {
		fmt.Fprintf(os.Stdout, "%v\n", err)
		return OK
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return UNKNOWN
	}

	err = opt.run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return CRITICAL
	}
	return OK
}
