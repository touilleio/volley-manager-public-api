const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { test } = require('node:test');
const vm = require('node:vm');

const malicious = '<img src=x onerror="alert(1)">: équipe';
const escape = value => String(value ?? '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');

function page(name) {
	let config;
	const handlers = [];
	const select = { options: [], appendChild(option) { this.options.push(option); } };
	const textRenderer = { display: escape, filter: escape };
	function DataTable(_, options) {
		config = options;
		this.on = () => {};
	}
	DataTable.render = { text: () => textRenderer };
	const context = vm.createContext({
		document: { getElementById: () => select },
		window: { location: { pathname: '/static/' + name + '.html' } },
		Option: function(text, value) { this.text = text; this.value = value; },
		DataTable,
		$: selector => {
			let text;
			return {
				ready: callback => callback(), each() {}, append() {}, toggleClass() {},
				on: (_, callback) => handlers.push(callback),
				find: () => ({ text: () => malicious, val: () => '42' }),
				text(value) { text = value; return this; }, html: () => escape(text),
			};
		},
	});
	context.$.getJSON = (_, callback) => callback({42: malicious});
	vm.runInContext(readFileSync('static/js/team-links.js', 'utf8'), context);
	context.bindTeamLinks = () => {};
	vm.runInContext(readFileSync('static/js/' + name + '.js', 'utf8'), context);
	if (name === 'ranking') handlers[0]();
	return { context, config, select, textRenderer };
}

for (const name of ['upcoming', 'past', 'ranking']) {
	test(name + ' preserves captions as option text and escapes table strings', () => {
		const { context, config, select, textRenderer } = page(name);
		assert.equal(select.options[1].text, malicious);
		assert.equal(select.options[1].value, '42', 'a colon in the caption must not replace the ID');
		for (const field of name === 'ranking' ? ['teamCaption'] : ['homeTeam', 'awayTeam', 'hall']) {
			assert.equal(config.columns.find(column => column.data === field).render, textRenderer);
		}
		if (name !== 'ranking') {
			assert.equal(context.renderLeagueCell(malicious, 'display', {isCup: false}), escape(malicious));
			const cup = context.renderLeagueCell(malicious, 'display', {isCup: true});
			assert.ok(cup.includes(escape(malicious)));
			assert.ok(!cup.includes('<img'));
		}
		if (name === 'past') {
			const winner = config.columns.find(column => column.data === 'winner').render;
			for (const side of ['home', 'away']) {
				assert.equal(winner('team_' + side, 'display', {homeTeam: malicious, awayTeam: malicious}), escape(malicious));
			}
		}
	});
}
