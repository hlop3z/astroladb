// restrict.js — Removes non-declarative globals from the JS runtime.
// Used for schema and migration contexts where only the DSL should be available.
// Generators get a fresh VM with full globals restored.
(function() {
	var restricted = [
		// Time
		'Date',
		// Math
		'Math', 'NaN', 'Infinity', 'isNaN', 'isFinite',
		'parseInt', 'parseFloat',
		// Collections
		'Map', 'Set', 'WeakMap', 'WeakSet',
		// Serialization
		'JSON',
		// Encoding
		'encodeURIComponent', 'decodeURIComponent',
		'encodeURI', 'decodeURI', 'escape', 'unescape',
		// Typed arrays
		'ArrayBuffer', 'DataView',
		'Int8Array', 'Uint8Array', 'Int16Array', 'Uint16Array',
		'Int32Array', 'Uint32Array', 'Float32Array', 'Float64Array',
		// Reflection
		'Symbol', 'Proxy', 'Reflect',
		// Other
		'WeakRef', 'console'
	];
	var blocked = 0;
	for (var i = 0; i < restricted.length; i++) {
		try {
			this[restricted[i]] = undefined;
			blocked++;
		} catch(e) {}
	}
	return blocked;
}).call(this);
