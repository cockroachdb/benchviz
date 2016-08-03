// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: William Haack (will@cockroachlabs.com)

// The purpose of filegenerator is to provide the tools for the automatic
// deployment of JSON files that are used on the benchviz web server
// as the data to be displayed. RenderHistoricalBenchmarkResults
// creates a BenchPackages struct that has all the data needed to
// generate most of the JSON files. The various other methods use
// the BenchPackages created by RenderHistoricalBenchmarkResults
// in order to create the JSON files.

package filegenerator

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	deployRootEnv    string = "BENCHDEPLOY"
	benchSamplesEnv  string = "BENCHSAMPLES"
	awsBucketNameEnv string = "AWSBUCKETNAME"
)

// BenchPackages is a mapping of package names to test names, which in turn map
// to their historical data.
type BenchPackages map[string]BenchTestMap

// BenchTestMap is a mapping of test names to dates, which in turn map to the
// results of the benchmark tests for that given date.
type BenchTestMap map[string]BenchResults

// BenchResults map dates to the benchmark test data for that date.
type BenchResults map[string]BenchStats

// BenchStats is a struct containing the various metrics obtained from running
// a benchmark test with the -benchmem parameter. The variable names are kept
// short to keep the JSON files small.
type BenchStats struct {
	N int     // ns/op
	A int     // alllocs/op
	B int     // B/op
	M float64 // MB/s
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func isValidTestLine(line string) bool {
	matched, err := regexp.MatchString("FAIL", line)
	check(err)
	// This matches a line from Stdout where a benchmark test passes.
	return len(line) >= 9 && line[0:9] == "Benchmark" && !matched
}

func parseTestResultLine(line string) (string, BenchStats) {
	testName := strings.Trim(strings.Split(line, "\t")[0], " ")
	nsopr := regexp.MustCompile("[0-9]+ ns/op")
	allocpr := regexp.MustCompile("[0-9]+ allocs/op")
	bopr := regexp.MustCompile("[0-9]+ B/op")
	mbpsr := regexp.MustCompile("[0-9]+\\.?[0-9]+ MB/s")
	var nsopi, allocpi, bopi int
	var mbpsf float64
	var err error
	if nsops := nsopr.FindString(line); nsops != "" {
		nsopi, err = strconv.Atoi(strings.Split(nsops, " ")[0])
		check(err)
	}
	if allocps := allocpr.FindString(line); allocps != "" {
		allocpi, err = strconv.Atoi(strings.Split(allocps, " ")[0])
		check(err)
	}
	if bops := bopr.FindString(line); bops != "" {
		bopi, err = strconv.Atoi(strings.Split(bops, " ")[0])
		check(err)
	}
	if mbpss := mbpsr.FindString(line); mbpss != "" {
		mbpsf, err = strconv.ParseFloat(strings.Split(mbpss, " ")[0], 64)
		check(err)
	}
	return testName, BenchStats{N: nsopi, A: allocpi, B: bopi, M: mbpsf}
}

func isValidDataDir(dataDirName string) bool {
	matched, err := regexp.MatchString("[0-9]{2}-[0-9]{2}-[0-9]{4}", dataDirName)
	check(err)
	return matched
}

func mustGetEnv(envName string) string {
	envVar := os.Getenv(envName)
	if envVar == "" {
		log.Fatalf("The env variable %s is not set.", envVar)
	}
	return envVar
}

// SyncWithAWS downloads the unorganized data from s3.
func SyncWithAWS() {
	benchSamples := mustGetEnv(benchSamplesEnv)
	awsBucketName := mustGetEnv(awsBucketNameEnv)
	cmd := exec.Command("aws", "s3", "sync", "s3://"+awsBucketName+"/benchHistoricalData", benchSamples)
	runWithStandardOutputs(cmd)
}

// RenderHistoricalBenchmarkResults takes the folders in benchSamples, parses them, and returns a
// data structure called BenchPackages which represent the data from the folders.
func RenderHistoricalBenchmarkResults(dirs []string) BenchPackages {
	dirToTestNames := make(BenchPackages)
	benchSamples := mustGetEnv(benchSamplesEnv)
	dataDirs, err := ioutil.ReadDir(benchSamples)
	check(err)
	for _, dataDir := range dataDirs {
		if !dataDir.IsDir() {
			continue
		}
		dataDirName := dataDir.Name()
		if !isValidDataDir(dataDirName) {
			continue
		}
		for _, dirName := range dirs {
			if _, ok := dirToTestNames[dirName]; !ok {
				dirToTestNames[dirName] = make(BenchTestMap)
			}
			fullDirPath := filepath.Join(benchSamples, dataDirName, "cockroach", dirName)
			files, err := ioutil.ReadDir(fullDirPath)
			if err != nil {
				continue
			}
			for _, file := range files {
				fileName := file.Name()
				matched, err := regexp.MatchString(`.*test\.stdout`, fileName)
				check(err)
				if matched {

					content, err := ioutil.ReadFile(filepath.Join(fullDirPath, fileName))
					check(err)
					for _, line := range strings.Split(string(content), "\n") {
						if isValidTestLine(line) {
							testName, stats := parseTestResultLine(line)
							if _, ok := dirToTestNames[dirName][testName]; !ok {
								dirToTestNames[dirName][testName] = make(BenchResults)
							}
							dirToTestNames[dirName][testName][dataDirName] = stats
						}
					}
				}
			}
		}
	}
	return dirToTestNames
}

// GenerateJSONFiles takes a BenchPackages and creates a json file for every test
// containing its benchmark results data over time.
func GenerateJSONFiles(packages BenchPackages) {
	deployRoot := mustGetEnv(deployRootEnv)
	for dir := range packages {
		for testName := range packages[dir] {
			testData := packages[dir][testName]
			fileFullName := filepath.Join(deployRoot, dir, testName+".json")
			testDataJSON, err := json.Marshal(&testData)
			check(err)
			check(ioutil.WriteFile(fileFullName, testDataJSON, 0644))
		}
	}
}

// GenerateTestNameJSONFile creates a JSON file that represents a map of
// directories to a list of all of the benchmark tests in that directory.
func GenerateTestNameJSONFile(packages BenchPackages) {
	deployRoot := mustGetEnv(deployRootEnv)
	testPackageToName := make(map[string][]string)
	for packageName := range packages {
		testPackageToName[packageName] = make([]string,
			len(testPackageToName[packageName]))
		for testName := range packages[packageName] {
			testPackageToName[packageName] = append(testPackageToName[packageName], testName)
		}
	}
	fileName := filepath.Join(deployRoot, "test_names.json")
	testNamesJSON, err := json.Marshal(&testPackageToName)
	check(err)
	check(ioutil.WriteFile(fileName, testNamesJSON, 0644))
}

func getDatesFromPackages(packages BenchPackages) []string {
	var dates []string
	dateSet := make(map[string]bool)
	for _, dir := range packages {
		for _, tests := range dir {
			for date := range tests {
				dateSet[date] = true
			}
		}
	}
	for date := range dateSet {
		dates = append(dates, date)
	}
	return dates
}

// CopyWWW copies the files in the www directory into the aws deploy directory
// to be deployed to s3.
func CopyWWW() {
	deployRoot := mustGetEnv(deployRootEnv)
	wd, err := os.Getwd()
	check(err)
	fileNames := []string{"common.js", "generate_benchmark_list.js",
		"generate_benchmark_plot.js", "index.html", "plot.html"}
	for _, fileName := range fileNames {
		cmd := exec.Command("cp", filepath.Join(wd, "www", fileName), deployRoot)
		runWithStandardOutputs(cmd)
	}
}

// PublishToAWS uploads the files in the aws deploy directory to the configured
// s3 instance.
func PublishToAWS() {
	deployRoot := mustGetEnv(deployRootEnv)
	awsBucketName := mustGetEnv(awsBucketNameEnv)
	cmd := exec.Command("aws", "s3", "sync", deployRoot, "s3://"+awsBucketName, "--acl", "public-read")
	runWithStandardOutputs(cmd)
}

func runWithStandardOutputs(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	check(cmd.Run())
}
