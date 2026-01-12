(function() {
	try {
		Object.freeze(Object.prototype);
		Object.freeze(Array.prototype);
		Object.freeze(String.prototype);
		Object.freeze(Number.prototype);
		Object.freeze(Boolean.prototype);
	} catch(e) {}
})();
