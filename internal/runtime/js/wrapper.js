(function() {
	var __result = {{CODE}};

	// Check if result is a chainable object (new API) with _getResult method
	if (__result && typeof __result._getResult === 'function') {
		__result = __result._getResult();
	}

	// Register the table if it has columns (table builder result)
	if (__result && __result.columns) {
		__registerTable("{{NAMESPACE}}", "{{TABLE_NAME}}", __result);
	}
})();
