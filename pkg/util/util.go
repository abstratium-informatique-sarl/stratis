package util

import (
	"bufio"
	"os/exec"
	"sync"
	"strings"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
)

var log = logging.GetLog("util")

func RunCommand(dir string, name string, args ...string) (*sync.WaitGroup, chan string, chan string, chan int, error) {
	log.Debug().Msgf("Running command: %s %s in directory: %s", name, strings.Join(args, " "), dir)
	
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	
	// Set environment variables that might help with output buffering
	cmd.Env = append(cmd.Env, "TERM=dumb", "COLUMNS=1000")
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create stdout pipe for %s", name)
		return nil, nil, nil, nil, err
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create stderr pipe for %s", name)
		return nil, nil, nil, nil, err
	}
	
	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msgf("Failed to start %s", name)
		return nil, nil, nil, nil, err
	}
	
	wg := sync.WaitGroup{}
	wg.Add(3)
	
	stdoutChan := make(chan string)
	stderrChan := make(chan string)
	rcChan := make(chan int)
	
	// Process stdout line by line
	go func() {
		defer close(stdoutChan)
		defer wg.Done()
		
		// Use a larger buffer size for the scanner
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		
		for scanner.Scan() {
			line := scanner.Text()
			stdoutChan <- line
		}
		
		if err := scanner.Err(); err != nil {
			e := err.Error()
			if e != "read |0: file already closed" {
				log.Error().Err(err).Msgf("Error reading stdout from %s", name)
				stdoutChan <- "Error reading stdout: " + e
			}
		}
	}()
	
	// Process stderr line by line
	go func() {
		defer close(stderrChan)
		defer wg.Done()
		
		scanner := bufio.NewScanner(stderr)
		buf := make([]byte, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		
		for scanner.Scan() {
			line := scanner.Text()
			stderrChan <- line
		}
		
		if err := scanner.Err(); err != nil {
			e := err.Error()
			if e != "read |0: file already closed" {
				log.Error().Err(err).Msgf("Error reading stderr from %s", name)
				stderrChan <- "Error reading stderr: " + e
			}
		}
	}()
	
	go func() {
		defer close(rcChan)
		defer wg.Done()
		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msgf("Failed to wait for %s", name)
		}
		log.Debug().Msgf("command completed: %s with exit code: %d", name, cmd.ProcessState.ExitCode())
		rcChan <- cmd.ProcessState.ExitCode()
	}()
	
	return &wg, stdoutChan, stderrChan, rcChan, nil
}

func RunAndWait(localPath string, name string, args ...string) (string, int, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = localPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), -999, err
	}
	return string(out), 0, nil
}