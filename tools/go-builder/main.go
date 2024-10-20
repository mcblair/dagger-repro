package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func main() {
	var cmdBuild = &cobra.Command{
		Use:   "build",
		Short: "Builds a Go project from a specified source directory and outputs to the specified directory",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			sourceDir := args[0]
			outputDir := args[1]

			// Construct the go build command with the specified output directory
			buildCmd := exec.Command("go", "build", "-o", outputDir, sourceDir)

			// Set the working directory to the source directory
			buildCmd.Dir = sourceDir

			// Run the command and capture the output
			output, err := buildCmd.CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing go build: %s\nOutput:\n%s\n", err, string(output))
				os.Exit(1)
			}

			fmt.Println("Build successful, binary created at:", outputDir)
		},
	}

	var rootCmd = &cobra.Command{Use: "gobuilder"}
	rootCmd.AddCommand(cmdBuild)
	rootCmd.Execute()
}
