const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { test } = require('node:test');
const vm = require('node:vm');

const malicious = '<img src=x onerror="alert(1)">: équipe';
const escape = value => String(value ?? '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');

function page(name, teamId = '42') {
	let config;
	const handlers = [];
	const select = { selectedIndex: 1, options: [], appendChild(option) { this.options.push(option); } };
	const filter = { children: [], appendChild(child) { this.children.push(child); } };
	const textRenderer = { display: escape, filter: escape };
	function DataTable(_, options) {
		config = options;
		this.on = () => {};
		this.ajax = { url: () => ({ load() {} }) };
		this.draw = () => {};
	}
	DataTable.render = { text: () => textRenderer };
	const context = vm.createContext({
		document: {
			getElementById: id => id === 'team-filter' ? filter : select,
			createElement: tag => ({ tagName: tag.toUpperCase() }),
		},
		window: { location: { pathname: '/static/' + name + '.html' } },
		Option: function(text, value) { this.text = text; this.value = value; },
		DataTable,
		$: (selector, attributes) => {
			if (selector === '<a>') {
				return { tagName: 'A', id: attributes.id, href: attributes.href,
					target: attributes.target, textContent: attributes.text };
			}
			let text;
			return {
				ready: callback => callback(), each() {}, toggleClass() {}, remove() {},
				append(child) {
					if (selector === '#team-filter' && typeof child === 'object') filter.appendChild(child);
				},
				on: (_, callback) => handlers.push(callback),
				find: () => ({
					text: () => select.options[select.selectedIndex].text,
					val: () => select.options[select.selectedIndex].value,
				}),
				text(value) { text = value; return this; }, html: () => escape(text),
			};
		},
	});
	context.$.getJSON = (_, callback) => callback({[teamId]: malicious});
	vm.runInContext(readFileSync('static/js/team-links.js', 'utf8'), context);
	context.bindTeamLinks = () => {};
	vm.runInContext(readFileSync('static/js/' + name + '.js', 'utf8'), context);
	if (name === 'ranking') handlers[0]();
	return { context, config, select, filter, handlers, textRenderer };
}

test('calendar export keeps selected values in the URL path and creates one link', () => {
	for (const teamId of ['42', '\"><img src=x onerror="alert(1)">', '../other?x=1#fragment']) {
		const { filter, handlers, select } = page('upcoming', teamId);
		handlers[0].call(select);
		assert.equal(filter.children.length, 1);
		const link = filter.children[0];
		assert.equal(link.tagName, 'A');
		assert.equal(link.id, 'ics_export');
		assert.equal(link.href, '/ics/upcoming/' + encodeURIComponent(teamId));
		assert.equal(link.target, '_blank');
		assert.equal(link.textContent, 'Exporter calendrier');
	}
	const { filter, handlers, select } = page('upcoming');
	select.selectedIndex = 0;
	handlers[0].call(select);
	assert.equal(filter.children.length, 0, 'the all-teams option must not create a calendar link');
});

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
