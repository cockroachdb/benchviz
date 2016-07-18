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
	"math"
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

// GeometricMeanData is a struct that contains the means of the benchmark
// stats for all the tests in a given package.
type GeometricMeanData struct {
	NMean float64 // Mean of the ns/op result from all tests in a package.
	AMean float64 // Mean of the allocs/op result from all tests in a package.
	BMean float64 // Mean of the B/op result from all tests in a package.
	MMean float64 // Mean of the MB/s result from all tests in a package.
	Date  string  // The date that the mean of all tests was collected.
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

// RenderHistoricalBenchmarkResults takes the folders in benchSamples, parses them, and returns a
// data structure called BenchPackages which represent the data from the folders.
func RenderHistoricalBenchmarkResults(dirs []string) BenchPackages {
	dirToTestNames := make(BenchPackages)
	benchSamples := os.Getenv(benchSamplesEnv)
	if benchSamples == "" {
		log.Fatalf("The env variable %s is not set.", benchSamplesEnv)
	}
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
	deployRoot := os.Getenv(deployRootEnv)
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
	deployRoot := os.Getenv(deployRootEnv)
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

// GenerateGeometricMeanJSONFile creates a JSON file that has the geometric mean of
// the results from every benchmark test in a package for every package.
func GenerateGeometricMeanJSONFile(packages BenchPackages, dirs []string) {
	deployRoot := os.Getenv(deployRootEnv)
	dates := getDatesFromPackages(packages)
	packageToGeometricMean := make(map[string][]GeometricMeanData)
	for _, dir := range dirs {
		packageToGeometricMean[dir] = make([]GeometricMeanData, 0)
		for _, date := range dates {
			vectors := make(map[string][]float64)
			for _, test := range packages[dir] {
				vectors["N"] = append(vectors["N"], float64(test[date].N))
				vectors["A"] = append(vectors["A"], float64(test[date].A))
				vectors["B"] = append(vectors["B"], float64(test[date].B))
				vectors["M"] = append(vectors["M"], float64(test[date].M))
			}
			nMean := GetGeometricMean(vectors["N"])
			aMean := GetGeometricMean(vectors["A"])
			bMean := GetGeometricMean(vectors["B"])
			mMean := GetGeometricMean(vectors["M"])
			packageToGeometricMean[dir] = append(packageToGeometricMean[dir],
				GeometricMeanData{NMean: nMean, AMean: aMean, BMean: bMean, MMean: mMean, Date: date})
		}
	}
	fileName := filepath.Join(deployRoot, "geometric_means.json")
	geometricMeansJSON, err := json.Marshal(&packageToGeometricMean)
	check(err)
	check(ioutil.WriteFile(fileName, geometricMeansJSON, 0644))
}

// GetGeometricMean returns the geometric mean of a vector.
func GetGeometricMean(vector []float64) float64 {
	epsilon := .0001
	sum := 0.0
	size := 0
	/// Use law of logs to avoid overflows when multiplying entire vector.
	for _, num := range vector {
		if num > epsilon {
			sum += math.Log(num)
			size++
		}
	}
	if size == 0 {
		return 0
	}
	return math.Exp(sum / float64(size))
}

// CopyWWW copies the files in the www directory into the aws deploy directory
// to be deployed to s3.
func CopyWWW() {
	deployRoot := os.Getenv(deployRootEnv)
	wd, err := os.Getwd()
	check(err)
	fileNames := []string{"common.js", "generate_benchmark_means.js", "generate_benchmark_list.js",
		"generate_benchmark_plot.js", "index.html", "geometric.html", "plot.html"}
	for _, fileName := range fileNames {
		cmd := exec.Command("cp", filepath.Join(wd, "www", fileName), deployRoot)
		runWithStandardOutputs(cmd)
	}
}

// PublishToAWS uploads the files in the aws deploy directory to the configured
// s3 instance.
func PublishToAWS() {
	deployRoot := os.Getenv(deployRootEnv)
	awsBucketName := os.Getenv(awsBucketNameEnv)
	cmd := exec.Command("aws", "s3", "sync", deployRoot, "s3://"+awsBucketName, "--acl", "public-read")
	runWithStandardOutputs(cmd)
}

func runWithStandardOutputs(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	check(cmd.Run())
}
