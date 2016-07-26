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

function loadJSON(fileName, callback) {
    var xobj = new XMLHttpRequest();
    xobj.overrideMimeType('application/json');
    xobj.open('GET', fileName, true);
    xobj.onreadystatechange = function () {
	if (xobj.readyState == 4 && xobj.status == '200') {
	    callback(xobj.responseText);
	}
    };
    xobj.send(null);
}

function getUrlVars() {
    var vars = [], hash;
    var hashes =window.location.href.slice(
	window.location.href.indexOf('?') + 1).split('&');
    for(var i = 0; i < hashes.length; i++)
    {
	hash = hashes[i].split('=');
	vars.push(hash[0]);
	vars[hash[0]] = hash[1];
    }
    return vars;
}

// Dates are in the format DD-MM-YYYY
sortDates = function(a, b) {
    a = a.split('-');
    b = b.split('-');
    if (a[2] != b[2]) {
	return a[2] - b[2]
    }
    if (a[1] != b[1]) {
	return a[1] - b[1];
    }
    return a[0] - b[0];
}
