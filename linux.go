package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
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

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func writeStats(filename string, ps processStats) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	jb, err := json.Marshal(ps)
	if err != nil {
		return err
	}
	_, err = file.Write(jb)
	return err
}

func readStats(filename string) (processStats, error) {
	ps := processStats{}
	d, err := os.ReadFile(filename)
	if err != nil {
		return ps, err
	}
	err = json.Unmarshal(d, &ps)
	if err != nil {
		return ps, err
	}
	return ps, nil
}

func cpuJiffer() (float64, error) {
	// read /proc/stat
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return 0, err
	}
	cpu, err := fs.Stat()
	if err != nil {
		return 0, err
	}
	return (cpu.CPUTotal.User + cpu.CPUTotal.Nice + cpu.CPUTotal.System + cpu.CPUTotal.Idle), nil
}

func (opt *Opt) fdsStat(p procfs.Proc, now uint64) error {
	fds, err := p.FileDescriptorsLen()
	if err != nil {
		return errors.Wrap(err, "Could not get fds")
	}

	limit, err := p.Limits()
	if err != nil {
		return errors.Wrap(err, "Could not get limits")
	}

	fmt.Printf("process-status.fds_%s.count\t%d\t%d\n", opt.KeyPrefix, fds, now)
	fmt.Printf("process-status.fds_%s.max\t%d\t%d\n", opt.KeyPrefix, limit.OpenFiles, now)
	fmt.Printf("process-status.fds_usage_%s.percentage\t%f\t%d\n", opt.KeyPrefix, float64(fds)*100/float64(limit.OpenFiles), now)

	return nil
}

func (opt *Opt) memStat(p procfs.Proc, now uint64) error {
	pss, err := p.Stat()
	if err != nil {
		return errors.Wrap(err, "Could not get NewStat")
	}
	used := pss.ResidentMemory()

	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return errors.Wrap(err, "Could not get NewDefaultFS")
	}
	ms, err := fs.Meminfo()
	if err != nil {
		return errors.Wrap(err, "Could not get getMemStat")
	}
	// XXX use MemTotal as max memory. not concern cgroup
	memTotal := ms.MemTotal
	max := *memTotal * 1024

	fmt.Printf("process-status.mem_%s.used\t%d\t%d\n", opt.KeyPrefix, used, now)
	fmt.Printf("process-status.mem_%s.max\t%d\t%d\n", opt.KeyPrefix, max, now)
	fmt.Printf("process-status.mem_usage_%s.percentage\t%f\t%d\n", opt.KeyPrefix, float64(used)*100/float64(max), now)
	return nil
}

func (opt *Opt) cpuStat(p procfs.Proc, now uint64) error {
	pss, err := p.Stat()
	if err != nil {
		return errors.Wrap(err, "Could not get process stat")
	}

	c, err := cpuJiffer()
	if err != nil {
		return errors.Wrap(err, "failed to fetch /proc/stat")
	}

	ps := processStats{
		Now:    now,
		SysCPU: c,
		CPU:    pss.CPUTime(),
	}

	workDir := pluginutil.PluginWorkDir()
	curUser, _ := user.Current()
	prevPath := filepath.Join(workDir, fmt.Sprintf("%s-process-status-v2-%s-%d", curUser.Uid, opt.KeyPrefix, p.PID))

	if !fileExists(prevPath) {
		err = writeStats(prevPath, ps)
		if err != nil {
			return errors.Wrap(err, "failed to save stats")
		}
		fmt.Fprintf(os.Stderr, "Notice: first time execution command\n")
		return nil
	}

	prev, err := readStats(prevPath)
	if err != nil {
		return errors.Wrap(err, "failed to load stats")
	}

	us := (float64(ps.CPU-prev.CPU) / float64(ps.SysCPU-prev.SysCPU)) * 100
	fmt.Printf("process-status.cpu_%s.percentage\t%f\t%d\n", opt.KeyPrefix, us, now)
	err = writeStats(prevPath, ps)
	if err != nil {
		return errors.Wrap(err, "failed to save stats")
	}

	return nil
}

func (opt *Opt) run() error {

	now := uint64(time.Now().Unix())

	proc, err := procfs.NewProc(opt.Pid)
	if err != nil {
		return errors.Wrap(err, "failed to fetch proc")
	}

	err = opt.fdsStat(proc, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Notice: %v\n", err)
	}

	err = opt.cpuStat(proc, now)
	if err != nil {
		return err
	}

	err = opt.memStat(proc, now)
	if err != nil {
		return err
	}

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
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	err = opt.run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}
