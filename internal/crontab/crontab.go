package crontab

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const maxBackups = 10

func backupDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "cronk", "backups"), nil
}

func writeBackup(content string) error {
	dir, err := backupDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := "crontab-" + time.Now().Format("20060102-150405")
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		return err
	}
	return pruneBackups(dir)
}

func pruneBackups(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "crontab-") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) <= maxBackups {
		return nil
	}
	for _, n := range names[:len(names)-maxBackups] {
		_ = os.Remove(filepath.Join(dir, n))
	}
	return nil
}

type Job struct {
	Raw      string
	Schedule string
	Command  string
	Disabled bool
	Reboot   bool
	parsed   cron.Schedule
}

func (j *Job) NextRun(from time.Time) (time.Time, bool) {
	if j.Reboot || j.parsed == nil {
		return time.Time{}, false
	}
	return j.parsed.Next(from), true
}

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

var ErrNoCrontabBinary = errors.New("crontab binary not found in PATH (install cron, e.g. on NixOS add `cron` to your packages)")

func ParseSchedule(s string) (cron.Schedule, error) {
	return parser.Parse(strings.TrimSpace(s))
}

func ValidateSchedule(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("empty")
	}
	if s == "@reboot" {
		return nil
	}
	_, err := parser.Parse(s)
	return err
}

func NextRunsFor(s string, n int, from time.Time) ([]time.Time, error) {
	sched, err := ParseSchedule(s)
	if err != nil {
		return nil, err
	}
	out := make([]time.Time, 0, n)
	t := from
	for i := 0; i < n; i++ {
		t = sched.Next(t)
		out = append(out, t)
	}
	return out, nil
}

func Load() ([]Job, error) {
	raw, err := LoadRaw()
	if err != nil {
		return nil, err
	}
	return parse(raw), nil
}

func LoadRaw() (string, error) {
	if _, err := exec.LookPath("crontab"); err != nil {
		return "", ErrNoCrontabBinary
	}
	cmd := exec.Command("crontab", "-l")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.ToLower(stderr.String())
		if msg == "" || strings.Contains(msg, "no crontab") {
			return "", nil
		}
		return "", errors.New(strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func Write(content string) error {
	if _, err := exec.LookPath("crontab"); err != nil {
		return ErrNoCrontabBinary
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if current, err := LoadRaw(); err == nil {
		if berr := writeBackup(current); berr != nil {
			return errors.New("backup failed: " + berr.Error())
		}
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}

func AppendJob(raw, line string) string {
	if raw != "" && !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	return raw + line + "\n"
}

func ReplaceLine(raw, oldLine, newLine string) string {
	lines := strings.Split(raw, "\n")
	for i, l := range lines {
		if l == oldLine {
			lines[i] = newLine
			return strings.Join(lines, "\n")
		}
	}
	return AppendJob(raw, newLine)
}

func parse(content string) []Job {
	var jobs []Job
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		disabled := false
		work := trimmed
		if strings.HasPrefix(trimmed, "#") {
			uncommented := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if looksLikeJob(uncommented) {
				disabled = true
				work = uncommented
			} else {
				continue
			}
		}
		j, ok := parseLine(work)
		if !ok {
			continue
		}
		j.Raw = line
		j.Disabled = disabled
		jobs = append(jobs, j)
	}
	return jobs
}

func looksLikeJob(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "@") {
		return true
	}
	fields := strings.Fields(s)
	if len(fields) < 6 {
		return false
	}
	for i := 0; i < 5; i++ {
		if !strings.ContainsAny(fields[i], "0123456789*") {
			return false
		}
	}
	return true
}

func parseLine(s string) (Job, bool) {
	if strings.HasPrefix(s, "@") {
		fields := strings.Fields(s)
		if len(fields) < 2 {
			return Job{}, false
		}
		schedule := fields[0]
		command := strings.TrimSpace(strings.TrimPrefix(s, schedule))
		j := Job{Schedule: schedule, Command: command}
		if schedule == "@reboot" {
			j.Reboot = true
			return j, true
		}
		sched, err := parser.Parse(schedule)
		if err != nil {
			return Job{}, false
		}
		j.parsed = sched
		return j, true
	}
	fields := strings.Fields(s)
	if len(fields) < 6 {
		return Job{}, false
	}
	schedule := strings.Join(fields[:5], " ")
	command := strings.Join(fields[5:], " ")
	sched, err := parser.Parse(schedule)
	if err != nil {
		return Job{}, false
	}
	return Job{Schedule: schedule, Command: command, parsed: sched}, true
}
