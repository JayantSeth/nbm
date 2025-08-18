package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
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
	TakeMultipleBackup(data.Nodes)
	fmt.Printf("Total time taken: %v\n", time.Since(start_time))
}