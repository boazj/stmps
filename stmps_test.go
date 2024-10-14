package main

import (
	"os"
	"runtime"
	"testing"

	"github.com/spezifisch/stmps/logger"
	"github.com/spezifisch/stmps/mpvplayer"
	"github.com/stretchr/testify/assert"
)

// Test initialization of the player
func TestPlayerInitialization(t *testing.T) {
	logger := logger.Init()
	player, err := mpvplayer.NewPlayer(logger)
	assert.NoError(t, err, "Player initialization should not return an error")
	assert.NotNil(t, player, "Player should be initialized")
}

func TestMainWithoutTUI(t *testing.T) {
	// Mock osExit to prevent actual exit during test
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true

		if code != 0 {
			// Capture and print the stack trace
			stackBuf := make([]byte, 1024)
			stackSize := runtime.Stack(stackBuf, false)
			stackTrace := string(stackBuf[:stackSize])

			// Print the stack trace with new lines only
			t.Fatalf("Unexpected exit with code: %d\nStack trace:\n%s\n", code, stackTrace)
		}
		// Since we don't abort execution here, we will run main() until the end or a panic.
	}
	headlessMode = true
	testMode = true

	// Restore patches after the test
	defer func() {
		osExit = os.Exit
		headlessMode = false
		testMode = false
	}()

	// Set command-line arguments to trigger the help flag
	os.Args = []string{"doesntmatter", "--config=stmp-example.toml"}

	main()

	if !exitCalled {
		t.Fatalf("osExit was not called")
	}
}

// Regression test for https://github.com/spezifisch/stmps/issues/70
func TestMainWithConfigFileEmptyString(t *testing.T) {
	// Mock osExit to prevent actual exit during test
	exitCalled := false
	osExit = func(code int) {
		exitCalled = true

		if code != 0x23420001 && code != 2 {
			// Capture and print the stack trace
			stackBuf := make([]byte, 1024)
			stackSize := runtime.Stack(stackBuf, false)
			stackTrace := string(stackBuf[:stackSize])

			// Print the stack trace with new lines only
			t.Fatalf("Unexpected exit with code: %d\nStack trace:\n%s\n", code, stackTrace)
		}
		// Since we don't abort execution here, we will run main() until the end or a panic.
	}
	headlessMode = true
	testMode = true

	// Restore patches after the test
	defer func() {
		osExit = os.Exit
		headlessMode = false
		testMode = false
	}()

	// Set command-line arguments to trigger the help flag
	os.Args = []string{"stmps"}

	main()

	if !exitCalled {
		t.Fatalf("osExit was not called")
	}
}
