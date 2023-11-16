package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"
)

type Structure struct {
	Years []Year `yaml:"years"`
}

type Year struct {
	Name string `yaml:"name"`
	Ue   []Ue   `yaml:"ue"`
}

type Ue struct {
	Name      string     `yaml:"name"`
	Resources []Resource `yaml:"resources"`
}

type Resource struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Volume string `yaml:"volume"`
	Url    string `yaml:"url"`
}

var filePaths []string

func main() {
	dir := "./parcours-hybridation"

	err := filepath.Walk(dir, walkDirectory)
	if err != nil {
		fmt.Printf("error walking the path %v: %v\n", dir, err)
	}

	var wg sync.WaitGroup
	resultCh := make(chan string)

	// Launch a goroutine for each file
	for _, filePath := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			readFile(filePath, &wg, resultCh)
		}(filePath)
	}

	// Close the channel after all goroutines are done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect and print results from the channel
	for result := range resultCh {
		fmt.Println(result)
	}
}

func walkDirectory(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err) // can't walk here,
		return nil       // ignore this error.
	}

	if info.IsDir() {
		return nil // not a file. ignore.
	}

	if filepath.Ext(info.Name()) == ".yml" {
		filePaths = append(filePaths, path)
	}

	return nil
}

func readFile(path string, wg *sync.WaitGroup, resultCh chan<- string) {
	fmt.Println(path)

	// Read the file
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	var structure Structure
	err = yaml.Unmarshal(yamlFile, &structure)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	for _, year := range structure.Years {
		for _, ue := range year.Ue {
			for _, resource := range ue.Resources {
				if isValidUrl(resource.Url) {
					wg.Add(1)
					go getHTTPStatus(resource.Url, wg, resultCh)
				}
			}
		}
	}
}

func isValidUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func getHTTPStatus(url string, wg *sync.WaitGroup, resultCh chan<- string) {
	defer wg.Done()

	// Send an HTTP GET request
	response, err := http.Get(url)
	if err != nil {
		resultCh <- fmt.Sprintf("%s: Error", url)
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		resultCh <- fmt.Sprintf("%s: %s", url, response.Status)
	}
}
