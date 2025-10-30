package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var (
	c config

	// Track the log file size and last handled error time so we only react to
	// new Sunshine errors.
	lastLogSize            int64
	lastMonitorMissingTime time.Time
)

// config controls how sunrise functions. See `sunrise.cfg.example` for comments
// on each item.
type config struct {
	SunriseCheckSeconds     int
	SunshineLogPath         string
	MonitorIsOffLogLine     string
	WakeMonitorSleepSeconds int
	StopSunshineCommand     string
	StartSunshineCommand    string
	WakeMonitorCommand      string
	EnableSunshineRestart   bool
}

func main() {
	configPath := flag.String("config", "/etc/sunrise/sunrise.cfg", "path to the sunrise config file")
	flag.Parse()

	_, err := toml.DecodeFile(*configPath, &c)
	if err != nil {
		log.Fatal("Error reading toml config file:", err)
	}

	log.Println("Starting sunrise monitoring service")

	ticker := time.NewTicker(time.Duration(c.SunriseCheckSeconds) * time.Second)
	for {
		<-ticker.C
		// Check to see if Sunshine logged that the monitor is missing
		monitorIsOff, err := isMonitorMissing()
		if err != nil {
			log.Fatal("Unable to read log file:", err)
		}
		// If the monitor is off...
		if monitorIsOff {
			// ...run the command to wake the monitor.
			err := wakeMonitor()
			if err != nil {
				log.Println("Could not wake monitor:", err)
			}
			waitForMonitor()
			if c.EnableSunshineRestart {
				err = restartSunshine()
			}
			if err != nil {
				log.Println("Could not restart sunshine:", err)
			}
		}
	}
}

// isMonitorMissing will search the current Sunshine log file for evidence that
// a client tried to connect to Sunshine and found the monitor was off. It
// returns `true` if the monitor is off.
func isMonitorMissing() (monitorIsMissing bool, err error) {
	log.Println("Checking if monitor is missing according to Sunshine log")
	logInfo, err := os.Stat(c.SunshineLogPath)
	if err != nil {
		return false, err
	}

	if logInfo.Size() < lastLogSize {
		// Sunshine rewrote the log, so the next matching line should trigger again.
		log.Println("Sunshine log appears to have rotated; resetting monitor-missing tracking state")
		resetMonitorTracking()
	}

	lastLogSize = logInfo.Size()

	logFile, err := os.Open(c.SunshineLogPath)
	if err != nil {
		return false, err
	}
	defer logFile.Close()

	var latestOccurrence time.Time

	// Walk the log to find the newest monitor-missing entry and capture its timestamp.
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, c.MonitorIsOffLogLine) {
			continue
		}

		entryTime, err := parseSunshineTimestamp(line)
		if err != nil {
			log.Printf("Unable to parse Sunshine log timestamp for line %q: %v", line, err)
			continue
		}

		if entryTime.After(latestOccurrence) {
			latestOccurrence = entryTime
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	if latestOccurrence.IsZero() {
		resetMonitorTracking()
		log.Println("Monitor is not missing")
		return false, nil
	}

	if lastMonitorMissingTime.IsZero() || latestOccurrence.After(lastMonitorMissingTime) {
		lastMonitorMissingTime = latestOccurrence
		log.Println("Monitor is missing; last Sunshine error at", latestOccurrence.Format(time.RFC3339Nano))
		return true, nil
	}

	log.Println("Monitor missing error already handled at", lastMonitorMissingTime.Format(time.RFC3339Nano))
	return false, nil
}

// wakeMonitor runs the configured command to wake the connected monitor from
// sleep.
func wakeMonitor() (err error) {
	wakeMonitorCommandAndArgs := strings.Split(c.WakeMonitorCommand, " ")
	wakeCMD := exec.Command(wakeMonitorCommandAndArgs[0], wakeMonitorCommandAndArgs[1:]...)
	log.Println("Running wakeMonitor command:", wakeCMD.String())
	err = wakeCMD.Run()
	if err != nil {
		return err
	}
	log.Println("wakeMonitor command completed without errors")
	return nil
}

// resetMonitorTracking resets the in-memory timestamp to track the last
// timestamp the monitor when missing.
func resetMonitorTracking() {
	lastMonitorMissingTime = time.Time{}
}

// parseSunshineTimestamp will obtain a Go-native timestamp out of the Sunshine
// logs.
func parseSunshineTimestamp(line string) (time.Time, error) {
	// Sunshine timestamps appear as: [YYYY-MM-DD HH:MM:SS.mmm]
	endIdx := strings.Index(line, "]")
	if !strings.HasPrefix(line, "[") || endIdx == -1 {
		return time.Time{}, fmt.Errorf("sunshine log line missing timestamp brackets")
	}

	timePortion := line[1:endIdx]
	t, err := time.ParseInLocation("2006-01-02 15:04:05.000", timePortion, time.Local)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// waitForMonitor will sleep for a configured amount of seconds for the monitor
// to wake up.
func waitForMonitor() {
	log.Println("Waiting", c.WakeMonitorSleepSeconds, "seconds for monitor to come up")
	time.Sleep(time.Duration(c.WakeMonitorSleepSeconds) * time.Second)
}

// restartSunshine will kill the existing sunshine process and start a new one.
// This isn't necessary in most cases.
func restartSunshine() (err error) {
	stopSunshine()
	err = startSunshine()
	if err != nil {
		return err
	}
	return nil
}

// stopSunshine will force-kill the existing sunshine service. If errors are
// encountered in this function, they are printed, but ignored. This is because
// the sunshine service may not be running, thus `killall` will exit with code
// `1`.
func stopSunshine() {
	stopSunshineCommandAndArgs := strings.Split(c.StopSunshineCommand, " ")
	stopSunshineCMD := exec.Command(stopSunshineCommandAndArgs[0], stopSunshineCommandAndArgs[1:]...)
	log.Println("Running stopSunshine command:", stopSunshineCMD.String())
	err := stopSunshineCMD.Run()
	if err != nil {
		log.Println("stopSunshine encountered an error - ignoring:", err)
	}
	log.Println("stopSunshine command completed without errors")
}

// startSunshine will start a new sunshine process.
func startSunshine() (err error) {
	startSunshineCommandAndArgs := strings.Split(c.StartSunshineCommand, " ")
	startSunshineCMD := exec.Command(startSunshineCommandAndArgs[0], startSunshineCommandAndArgs[1:]...)
	log.Println("Running startSunshine command:", startSunshineCMD.String())
	err = startSunshineCMD.Start()
	if err != nil {
		return err
	}
	log.Println("startSunshine command completed without errors")
	return nil
}
