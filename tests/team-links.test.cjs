// Run with: node --test tests/team-links.test.cjs
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { test } = require('node:test');
const vm = require('node:vm');

test('team links select loaded teams and keep the URL and filter in sync', () => {
	const select = {
		selectedIndex: 0,
		options: [
			{ text: 'Toutes', value: '' },
			{ text: 'H2', value: '42' },
			{ text: 'Équipe F1', value: '73' },
		],
	};
	const location = { pathname: '/upcoming.html', search: '?view=compact', hash: '#h2' };
	const changes = [];
	const history = [];
	const handlers = [() => changes.push(select.options[select.selectedIndex].value)];
	const events = {};
	const context = vm.createContext({
		document: { getElementById: () => select },
		window: {
			location,
			history: { pushState: (_, __, url) => {
				history.push(url);
				location.hash = new URL(url, 'https://example.test').hash;
			} },
			addEventListener: (event, handler) => { events[event] = handler; },
		},
		$: () => ({
			on: (_, handler) => handlers.push(handler),
			trigger: () => handlers.forEach(handler => handler()),
		}),
	});
	vm.runInContext(readFileSync('static/js/team-links.js', 'utf8'), context);
	context.bindTeamLinks();
	assert.equal(select.selectedIndex, 1);
	assert.deepEqual(changes, ['42']);
	assert.deepEqual(history, []);

	select.selectedIndex = 2;
	handlers.forEach(handler => handler());
	assert.equal(history.at(-1), '/upcoming.html?view=compact#%C3%A9quipe%20f1');

	for (const [hash, expected] of [
		['#H2', 1], ['#73', 2], ['#h2', 1],
		['#%C3%89quipe%20F1', 2], ['#unknown', 0],
		['#h2', 1], ['#%', 0], ['#h2', 1], ['', 0],
	]) {
		location.hash = hash;
		events.hashchange();
		assert.equal(select.selectedIndex, expected, hash);
	}
	assert.equal(history.length, 1, 'hash navigation must not add history entries');
	const count = changes.length;
	events.hashchange();
	assert.equal(changes.length, count, 'unchanged selection must not reload');

	select.selectedIndex = 1;
	handlers.forEach(handler => handler());
	select.selectedIndex = 0;
	handlers.forEach(handler => handler());
	assert.equal(history.at(-1), '/upcoming.html?view=compact');
});
