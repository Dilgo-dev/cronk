package crontab

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

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

func Load() ([]Job, error) {
	if _, err := exec.LookPath("crontab"); err != nil {
		return nil, ErrNoCrontabBinary
	}
	cmd := exec.Command("crontab", "-l")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.ToLower(stderr.String())
		if msg == "" || strings.Contains(msg, "no crontab") {
			return nil, nil
		}
		return nil, errors.New(strings.TrimSpace(stderr.String()))
	}
	return parse(stdout.String()), nil
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
		sched, err := parser.Parse(s[:len(schedule)])
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
