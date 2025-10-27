package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

var c config

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
			err = restartSunshine()
			if err != nil {
				log.Println("Could not restart sunshine:", err)
			}
		}
	}
}

// isMonitorMissing will search the current Sunshine log file (which is
// re-created from scratch each time Sunshine starts). It returns `true` if the
// monitor is off.
func isMonitorMissing() (monitorIsMissing bool, err error) {
	log.Println("Checking if monitor is missing according to Sunshine log")
	logBytes, err := os.ReadFile(c.SunshineLogPath)
	if err != nil {
		return false, err
	}

	if strings.Contains(string(logBytes), c.MonitorIsOffLogLine) {
		log.Println("Monitor is missing")
		return true, nil
	}

	log.Println("Monitor is not missing")
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

// waitForMonitor will sleep for a configured amount of seconds for the monitor
// to wake up.
func waitForMonitor() {
	log.Println("Waiting", c.WakeMonitorSleepSeconds, "seconds for monitor to come up")
	time.Sleep(time.Duration(c.WakeMonitorSleepSeconds) * time.Second)
}

// restartSunshine will kill the existing sunshine process and start a new one.
// This is necessary because waking the monitor up leaves sunshine in a weird
// state where the server won't start. The easiest solution is to restart the
// service.
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
