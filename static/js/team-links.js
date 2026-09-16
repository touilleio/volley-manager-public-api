function bindTeamLinks() {
	var select = document.getElementById('sel_team_id');
	var applyingHash = false;

	function applyHash() {
		var team;
		try {
			team = decodeURIComponent(window.location.hash.slice(1)).toLowerCase();
		} catch (error) {
			team = '';
		}
		var index = Array.from(select.options).findIndex(function(option, i) {
			return i > 0 && (option.text.toLowerCase() === team || option.value === team);
		});
		index = Math.max(0, index);
		if (select.selectedIndex !== index) {
			select.selectedIndex = index;
			applyingHash = true;
			$(select).trigger('change');
			applyingHash = false;
		}
	}

	$(select).on('change', function() {
		if (!applyingHash) {
			var hash = select.selectedIndex > 0
				? '#' + encodeURIComponent(select.options[select.selectedIndex].text.toLowerCase()) : '';
			if (window.location.hash !== hash) {
				window.history.pushState(null, '', window.location.pathname + window.location.search + hash);
			}
		}
	});
	window.addEventListener('hashchange', applyHash);
	applyHash();
}
