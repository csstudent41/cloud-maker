"use strict";
"use warnings";

function checkAll(checked, scope) {
	var inputs = scope.getElementsByTagName('input');
	for (var i = 0; i < inputs.length; i++) {
		if (inputs[i].type.toLowerCase() == 'checkbox') {
			inputs[i].checked = checked;
		}
	}
}

function toggleAll(checkbox) {
	var scope = document.getElementById('file-list')
	if (checkbox.checked) {
		checkAll("checked", scope);
	} else {
		checkAll("", scope);
	}
}

function clear_newdir_bg(div) {
	document.getElementById('newdir-cb').checked = "";
}
