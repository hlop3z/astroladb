(function() {
	var __result = {{CODE}};

	// Check if result is a chainable object (new API) with _getResult method
	if (__result && typeof __result._getResult === 'function') {
		__result = __result._getResult();
	}

	// Register the table if it has a builder reference
	if (__result && __result._tableBuilder) {
		__registerTable("{{NAMESPACE}}", "{{TABLE_NAME}}", __result);
	}
})();
