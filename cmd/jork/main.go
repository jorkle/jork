package main

import (
	\"fmt\"
	\"log\"
	\"os\"
	\"os/signal\"
	\"syscall\"

	\"github.com/jorkle/jork/internal/app\"
)

func main() {
	// Create the application
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf(\"Failed to create application: %v\", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start cleanup goroutine
	go func() {
		<-sigChan
		fmt.Println(\"\\nShutting down...\")
		if err := application.Cleanup(); err != nil {
			log.Printf(\"Error during cleanup: %v\", err)
		}
		os.Exit(0)
	}()

	// Run the application
	if err := application.Run(); err != nil {
		log.Fatalf(\"Application error: %v\", err)
	}

	// Cleanup on normal exit
	if err := application.Cleanup(); err != nil {
		log.Printf(\"Error during cleanup: %v\", err)
	}
}
