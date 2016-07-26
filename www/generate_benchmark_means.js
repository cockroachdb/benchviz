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

function loadGeometricMeanListFromJSON(callback) {
    loadJSON("geometric_means.json", callback);
}

function drawCharts(meanList) {
    var NPlotData = new google.visualization.DataTable();
    var APlotData = new google.visualization.DataTable();
    var BPlotData = new google.visualization.DataTable();
    var MPlotData = new google.visualization.DataTable();
    var plotData = [NPlotData, APlotData, BPlotData, MPlotData]
    $.each(plotData, function (index, plotData) {
	plotData.addColumn('string', 'Date');
	plotData.addColumn('number', 'value');
    });
    $.each(meanList, function (index, value) {
	NPlotData.addRow([value.Date, value.NMean]);
	APlotData.addRow([value.Date, value.AMean]);
	BPlotData.addRow([value.Date, value.BMean]);
	MPlotData.addRow([value.Date, value.MMean]);
    });
    var titleToPlotAndData = {
	'ns/op': ['NPlot', NPlotData],
	'allocs/op': ['APlot', APlotData],
	'B/op': ['BPlot', BPlotData],
	'MB/s': ['MPlot', MPlotData]
    }
    $.each(titleToPlotAndData, function (key, value){
	var options = {
	    title: key,
	    curveType: 'function',
	    legend: { position: 'bottom' }
	};
	new google.visualization.LineChart(document.getElementById(value[0])).draw(value[1], options);
    });
}

function initGoogleChart(dirName, dirNameToMeanList) {
    google.charts.load('current', {'packages':['corechart']});
    google.charts.setOnLoadCallback(function() {
	displayBenchMeans(dirName, dirNameToMeanList);
    });
}

function displayBenchMeans(dirName, dirNameToMeanList) {
    var sortedMeanList = dirNameToMeanList[dirName].sort(function(a, b) {
	return sortDates(a.Date, b.Date);
    });
    drawCharts(sortedMeanList);
}

function init() {
    loadGeometricMeanListFromJSON(function (response) {
	var dirNameToMeanList = JSON.parse(response);
	var dirName = getUrlVars()['directory'];
	$("#dirName").html(dirName);
	initGoogleChart(dirName, dirNameToMeanList);
    });
}

init();
