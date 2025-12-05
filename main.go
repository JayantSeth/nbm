package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	OUT_DIR = "./"
)

func main() {
	start_time := time.Now()
	yamlFile, err := os.ReadFile("data.yaml")
	if err != nil {
		log.Fatalf("Failed to open data file, is it present ?\n")
	}
	expandedContent := os.ExpandEnv(string(yamlFile))
	var data Data
	err = yaml.Unmarshal([]byte(expandedContent), &data)
	if err != nil {
		log.Fatalf("Failed to unmarshal: %s\n", err.Error())
	}
	// Create folder if it does not exist
	err = os.MkdirAll(data.OutDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create backup folder name: %s, error: %s\n", data.OutDir, err.Error())
	}
	OUT_DIR = data.OutDir
	TakeMultipleBackup(data.Nodes)
	fmt.Printf("Total time taken: %v\n", time.Since(start_time))
}
