(function() {
	var frozen = 0;
	function safeFr(obj) {
		try { Object.freeze(obj); frozen++; } catch(e) {}
	}
	safeFr(Object.prototype);
	safeFr(Array.prototype);
	safeFr(String.prototype);
	safeFr(Number.prototype);
	safeFr(Boolean.prototype);
	safeFr(RegExp.prototype);
	safeFr(Date.prototype);
	safeFr(Function.prototype);
	safeFr(Error.prototype);
	if (typeof Map !== 'undefined') safeFr(Map.prototype);
	if (typeof Set !== 'undefined') safeFr(Set.prototype);
	return frozen;
})();
