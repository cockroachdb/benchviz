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

function getTestDataFromJSON(directory, test, callback) {
    fileName = directory + '/' + test + '.json'
    loadJSON(fileName, callback);
}

function loadTestListFromJSON(callback) {
    loadJSON('test_names.json', callback);
}

function getResultsFromTestData(dates, testData) {
    var results = []
    for (var dateIndex in dates) {
	results.push({
	    Date: dates[dateIndex],
	    N: testData[dates[dateIndex]]['N'],
	    A: testData[dates[dateIndex]]['A'],
	    B: testData[dates[dateIndex]]['B'],
	    M: testData[dates[dateIndex]]['M']
	});
    }
    return results;
}

function getSortedDates(testData) {
    var dates = new Array();
    for (var date in testData) {
	dates.push(date);
    }
    dates.sort(sortDates);
    return dates
}

function ourDateToGoogDate(date) {
    var dateSplit = date.split('-');
    return new Date(parseInt(dateSplit[2]),
			   parseInt(dateSplit[1]) - 1,
			   parseInt(dateSplit[0]));
}

function drawCharts(results) {
    var testName = getUrlVars()['test'];
    NPlotData = new google.visualization.DataTable();
    APlotData = new google.visualization.DataTable();
    BPlotData = new google.visualization.DataTable();
    MPlotData = new google.visualization.DataTable();
    var plotData = [NPlotData, APlotData, BPlotData, MPlotData]
    $.each(plotData, function (index, plotData) {
	plotData.addColumn('date', 'Date');
	plotData.addColumn('number', testName);
    });
    $.each(results, function (index, value) {
	date = ourDateToGoogDate(value.Date);
	NPlotData.addRow([date, value.N]);
	APlotData.addRow([date, value.A]);
	BPlotData.addRow([date, value.B]);
	MPlotData.addRow([date, value.M]);
    });
    titleToPlotAndData = {
	'ns/op': ['NPlot', NPlotData],
	'allocs/op': ['APlot', APlotData],
	'B/op': ['BPlot', BPlotData],
	'MB/s': ['MPlot', MPlotData]
    }
    graphs = {};
    $.each(titleToPlotAndData, function (key, value){
	var options = {
	    title: key,
	    curveType: 'function',
	    vAxis: {minValue: 0},
	    legend: { position: 'bottom' },
	};
	var graph = new google.visualization.LineChart(document.getElementById(value[0]))
	graph.draw(value[1], options);
	graphs[value[0]] = graph;
    });
}

function initGraphs() {
    var urlVars = getUrlVars();
    $('#testName').html(urlVars['test']);
    getTestDataFromJSON(
	urlVars['directory'], urlVars['test'], function(response) {
	    var testData = JSON.parse(response);
	    var sortedDates = getSortedDates(testData);
	    var results = getResultsFromTestData(sortedDates, testData);
	    initGoogleCharts(results);
	});
}

function populateCompareToSelect() {
    var tests = [];
    loadTestListFromJSON(function(response) {
	var dirNameToTestList = JSON.parse(response);
	$.each(dirNameToTestList, function(directory, testList) {
	    testList.sort();
	    $.each(testList, function(index, test) {
		var optionValue = {
		    directory,
		    test
		}
		$('#compareSelect').append($('<option>', {
		    value: JSON.stringify(optionValue),
		    text: test
		}));
	    });
	});
	$('#compareSelect').chosen({search_contains: true});
    });
}

// Generates a new row for a data point where hte current graph
// does not have a data point for the givend ate.
function generateNewRow(plotData, googDate, value, colIndex) {
    var newRow = [googDate];
    for (var c = 1; c < plotData.getNumberOfColumns(); c++) {
	if (c == colIndex) {
	    newRow.push(value);
	} else {
	    newRow.push(null);
	}
    }
    plotData.addRow(newRow);
}

// Since certain tests are missing values for certain dates, we need to
// use this method in order to create a chart when we compare tests.
// Invariant: a chart only has one row per date.
function combineCurrentDateListWithResults(plotData, plotDivName, results, colIndex) {
    $.each(results, function(index, result) {
	var plotDivNameToResult = {
	    'NPlot': result.N,
	    'APlot': result.A,
	    'BPlot': result.B,
	    'MPlot': result.M  
	}
	var resultValue = plotDivNameToResult[plotDivName];
	var googDate = ourDateToGoogDate(result.Date);
	var row = plotData.getFilteredRows([{
	    column: 0,
	    value: googDate}]);
	if (row .length == 0) {
	    generateNewRow(plotData, googDate, resultValue, colIndex);
				
	} else {
	    plotData.setCell(row[0], colIndex, resultValue);
	}
    });
}

function addResultsToCurrentCharts(testName, results) {
    $.each(titleToPlotAndData, function(key, value) {
	var plotDivName = value[0];
	var plotData = value[1];
	var colIndex = plotData.addColumn('number', testName);
	combineCurrentDateListWithResults(plotData, plotDivName, results, colIndex);
	var options = {
	    title: key,
	    curveType: 'function',
	    vAxis: {minValue: 0},
	    legend: { position: 'bottom' }
	};
	graphs[plotDivName].draw(plotData, options);
    });
}

function setSelectOnClickHandler() {
    $('#compareSelect').on('change', function() {
	var testNameData = JSON.parse(this.value);
	var testName = testNameData.test
	var directory = testNameData.directory
	getTestDataFromJSON(directory, testName, function(response) {
	    var testData = JSON.parse(response);
	    var sortedDates = getSortedDates(testData);
	    var results = getResultsFromTestData(sortedDates, testData);
	    addResultsToCurrentCharts(testName, results);
	});
    });
}

function initGoogleCharts(results) {
    google.charts.load('current', {'packages':['corechart']});
    google.charts.setOnLoadCallback(function() {
	drawCharts(results);
    });
}

initGraphs();
populateCompareToSelect();
setSelectOnClickHandler();
