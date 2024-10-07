"use strict";
"use warnings";

function checkAll(checked, scope) {
	let inputs = scope.getElementsByTagName('input');
	for (var i = 0; i < inputs.length; i++) {
		if (inputs[i].type.toLowerCase() == 'checkbox') {
			inputs[i].checked = checked;
		}
	}
}

function toggleAll(checkbox) {
	let scope = document.getElementById('file-list')
	if (checkbox.checked) {
		checkAll(true, scope);
	} else {
		checkAll(false, scope);
	}
}

function deleteAction(button) {
	let table = document.getElementById('file-list')
	let tbody = table.getElementsByClassName('tbody')[0]
	let inputs = tbody.getElementsByTagName('input')
	let msg = 0;
	for (var i = 0; i < inputs.length; i++) {
		inputs[i].type.toLowerCase() == 'checkbox' &&
			inputs[i].checked && msg++;
	}
	if (msg == 0) {
		msg = "Whole Folder will be deleted."
	} else {
		msg += " file(s) selected."
	}
	if (confirm(msg + " Confirm Deletion?")) {
		button.formAction = "?action=delete";
		button.onclick = "submit()";
		button.click();
	}
}
