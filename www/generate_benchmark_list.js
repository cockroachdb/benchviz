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

function loadTestListFromJSON(callback) {
    loadJSON("test_names.json", callback);
}

function generateTestListHTML(directory, tests) {
    tests.sort();
    return _.values(tests).map((v) => `<a class="testName" href="/plot.html?directory=${directory}&test=${v}">${v}</a>`).join("<br/>");
}

function getDirectoryList(dirNameToTestList) {
    directoryList = ["sql", "sql/parser"];
    for (key in dirNameToTestList) {
	if (key != "sql" && key != "sql/parser") {
	    directoryList.push(key);
	}
    }
    return directoryList;
}

function populateList(dirs, dirNameToTestList) {
    var $template = $('.template');
    $template.hide()
    html = "";
    $.each(_.values(dirs), function (index, dir) {
	html += dir + '<br/>';
	var tests = _.values(dirNameToTestList[dir]);
	var testsHTML = generateTestListHTML(dir, tests);
	html += testsHTML + "<br/>";	
    });
    $('#testList').html(html);
}

function init() {
    loadTestListFromJSON(function(response) {
	var dirNameToTestList = JSON.parse(response);
	var dirs = getDirectoryList(dirNameToTestList);
	populateList(dirs, dirNameToTestList);
    });
}

init();
