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

package filegenerator_test

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/benchviz/filegenerator"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func TestGetGeometricMean(t *testing.T) {
	epsilon := .00001
	testVector := []float64{3, 9, 27}
	result := filegenerator.GetGeometricMean(testVector)
	if math.Abs(result-9.0) > epsilon {
		t.Errorf("Did not properly calculate geometric mean")
	}
	testVector = []float64{0, 9, 27}
	result = filegenerator.GetGeometricMean(testVector)
	if math.Abs(result-15.58846) > epsilon {
		t.Errorf("Did not properly calculate geometric mean")
	}
}

func setupSampleBenchDir() {
	sqlDir := filepath.Join(os.TempDir(), "benchSamples", "01-01-2015", "cockroach", "sql")
	check(os.MkdirAll(sqlDir, 0755))
	fileText := []byte("BenchmarkBank2_Cockroach-8 \t 1000 \t 1328086 ns/op \t 183965 B/op \t 2317 allocs/op")
	f, err := os.Create(filepath.Join(sqlDir, "sql.test.stdout"))
	check(err)
	_, _ = f.Write(fileText)
	check(f.Sync())
}

func removeSampleBenchDir() {
	check(os.RemoveAll(filepath.Join(os.TempDir(), "benchSamples")))
}

func TestRenderHistoricalBenchmarkResults(t *testing.T) {
	_ = os.Setenv("BENCHSAMPLES", filepath.Join(os.TempDir(), "benchSamples"))
	setupSampleBenchDir()
	defer removeSampleBenchDir()
	dirs := []string{"sql"}
	bPackages := filegenerator.RenderHistoricalBenchmarkResults(dirs)
	stats := bPackages["sql"]["BenchmarkBank2_Cockroach-8"]["01-01-2015"]
	if stats.N != 1328086 {
		t.Errorf("Did not properly obtain ns/op")
	} else if stats.B != 183965 {
		t.Errorf("Did not properly obtain B/op")
	} else if stats.A != 2317 {
		t.Errorf("Did not properly obtain allocs/op")
	} else if stats.M != 0 {
		t.Errorf("Did not properly obtain MB/s")
	}
}

func setupAWSDeployDir() {
	check(os.MkdirAll(filepath.Join(os.TempDir(), "awsDeploy", "sql"), 0755))
}

func removeAWSDeployDir() {
	check(os.RemoveAll(filepath.Join(os.TempDir(), "awsDeploy")))
}

func TestGenerateJSONFiles(t *testing.T) {
	stats := filegenerator.BenchStats{N: 1, A: 2, B: 3, M: 4.0}
	results := filegenerator.BenchResults{"01-01-2015": stats}
	bmap := filegenerator.BenchTestMap{"BenchmarkSqlSampleTest": results}
	bpackage := filegenerator.BenchPackages{"sql": bmap}
	check(os.Setenv("BENCHDEPLOY", filepath.Join(os.TempDir(), "awsDeploy")))
	setupAWSDeployDir()
	defer removeAWSDeployDir()
	filegenerator.GenerateJSONFiles(bpackage)
	content, err := ioutil.ReadFile(filepath.Join(os.TempDir(), "awsDeploy", "sql", "BenchmarkSqlSampleTest.json"))
	if err != nil {
		t.Errorf("Couldn't load JSON file")
	}
	actual, _ := json.Marshal(results)
	if string(content) != string(actual) {
		t.Errorf("JSON file did not match the data")
	}

}

func TestGeometricMeanJSONFile(t *testing.T) {
	epsilon := .00001
	dirs := []string{"sql"}
	stats1 := filegenerator.BenchStats{N: 1, A: 2, B: 3, M: 0}
	stats2 := filegenerator.BenchStats{N: 1, A: 4, B: 9, M: 0}
	stats3 := filegenerator.BenchStats{N: 1, A: 8, B: 27, M: 0}
	results1 := filegenerator.BenchResults{"01-01-2015": stats1}
	results2 := filegenerator.BenchResults{"01-01-2015": stats2}
	results3 := filegenerator.BenchResults{"01-01-2015": stats3}
	bmap := filegenerator.BenchTestMap{"BenchmarkSqlSampleTest1": results1, "BenchmarkSqlSampleTest2": results2, "BenchmarkSqlSampleTest3": results3}
	bpackage := filegenerator.BenchPackages{"sql": bmap}
	check(os.Setenv("BENCHDEPLOY", filepath.Join(os.TempDir(), "awsDeploy")))
	setupAWSDeployDir()
	defer removeAWSDeployDir()
	filegenerator.GenerateGeometricMeanJSONFile(bpackage, dirs)
	content, err := ioutil.ReadFile(filepath.Join(os.TempDir(), "awsDeploy", "geometric_means.json"))
	if err != nil {
		t.Errorf("Couldn't load JSON file")
	}
	var result map[string][]filegenerator.GeometricMeanData
	err = json.Unmarshal(content, &result)
	if err != nil {
		t.Errorf("Couldn't load json")
	}
	if result["sql"][0].Date != "01-01-2015" {
		t.Errorf("Wrong date")
	}
	if math.Abs(result["sql"][0].NMean-1.0) > epsilon {
		t.Errorf("Wrong ns/op mean value")
	}
	if math.Abs(result["sql"][0].AMean-4.0) > epsilon {
		t.Errorf("Wrong alloc/op mean value")
	}
	if math.Abs(result["sql"][0].BMean-9.0) > epsilon {
		t.Errorf("Wrong B/op mean value")
	}
	if math.Abs(result["sql"][0].MMean-0.0) > epsilon {
		t.Errorf("Wrong MB?s mean value")
	}
}
