package wrapexec

import (
	"bufio"
	"fmt"
	"os/exec"
)

// modelled after https://github.com/nextdns/nextdns/blob/c8ef0d420907aad43996315eea16ef57e07080be/host/service/internal/run_unix.go
// http://zetcode.com/golang/exec-command/
// https://www.yellowduck.be/posts/reading-command-output-line-by-line/
func StartWrappedCommand(prefix string, name string, arg ...string) (*exec.Cmd, chan struct{}, error) {
	cmd := exec.Command(name, arg...)

	// Get a pipe to read from standard out
	r, _ := cmd.StdoutPipe()

	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans r in a line-by-line fashion
	scanner := bufio.NewScanner(r)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Sprintf("%s%s\n", prefix, line)
		}

		// We're all done, unblock the channel
		done <- struct{}{}
	}()

	// Start the command and check for errors
	err := cmd.Start()

	return cmd, done, err
}

func RunWrappedCommand(prefix string, name string, arg ...string) (*exec.Cmd, error) {
	cmd := exec.Command(name, arg...)

	// Get a pipe to read from standard out
	r, _ := cmd.StdoutPipe()

	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})

	// Create a scanner which scans r in a line-by-line fashion
	scanner := bufio.NewScanner(r)

	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("%s%s\n", prefix, line)
		}

		// We're all done, unblock the channel
		done <- struct{}{}
	}()

	// Start the command and check for errors
	err := cmd.Run()

	// Wait for all output to be processed
	<-done

	return cmd, err
}
