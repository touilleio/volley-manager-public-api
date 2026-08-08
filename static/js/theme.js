(function () {
	var storageKey = "gibloux-theme";
	var darkQuery = "(prefers-color-scheme: dark)";
	var root = document.documentElement;
	var mediaQuery = window.matchMedia ? window.matchMedia(darkQuery) : null;

	function isTheme(value) {
		return value === "light" || value === "dark";
	}

	function storedTheme() {
		try {
			var value = window.localStorage.getItem(storageKey);
			return isTheme(value) ? value : null;
		} catch (error) {
			return null;
		}
	}

	function saveTheme(theme) {
		try {
			window.localStorage.setItem(storageKey, theme);
		} catch (error) {
			return;
		}
	}

	function systemTheme() {
		return mediaQuery && mediaQuery.matches ? "dark" : "light";
	}

	function currentTheme() {
		return root.getAttribute("data-theme") === "dark" ? "dark" : "light";
	}

	function applyTheme(theme) {
		root.setAttribute("data-theme", theme);
		root.style.colorScheme = theme;
	}

	function updateToggle(toggle) {
		var isDark = currentTheme() === "dark";
		var label = isDark ? "Mode sombre" : "Mode clair";
		toggle.setAttribute("aria-pressed", isDark ? "true" : "false");
		toggle.setAttribute("aria-label", isDark ? "Activer le mode clair" : "Activer le mode sombre");
		var labelElement = toggle.querySelector("[data-theme-label]");
		if (labelElement) {
			labelElement.textContent = label;
		}
		var sunIcon = toggle.querySelector('[data-theme-icon="sun"]');
		var moonIcon = toggle.querySelector('[data-theme-icon="moon"]');
		if (sunIcon) {
			sunIcon.hidden = isDark;
		}
		if (moonIcon) {
			moonIcon.hidden = !isDark;
		}
	}

	function updateToggles() {
		var toggles = document.querySelectorAll("[data-theme-toggle]");
		for (var index = 0; index < toggles.length; index += 1) {
			updateToggle(toggles[index]);
		}
	}

	var manualTheme = storedTheme();
	var hasManualTheme = isTheme(manualTheme);
	applyTheme(hasManualTheme ? manualTheme : systemTheme());

	function wireToggle(toggle) {
		updateToggle(toggle);
		toggle.addEventListener("click", function () {
			var nextTheme = currentTheme() === "dark" ? "light" : "dark";
			hasManualTheme = true;
			applyTheme(nextTheme);
			saveTheme(nextTheme);
			updateToggles();
		});
	}

	function ready() {
		var toggles = document.querySelectorAll("[data-theme-toggle]");
		for (var index = 0; index < toggles.length; index += 1) {
			wireToggle(toggles[index]);
		}
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", ready);
	} else {
		ready();
	}

	if (mediaQuery) {
		var systemChange = function () {
			if (!hasManualTheme) {
				applyTheme(systemTheme());
				updateToggles();
			}
		};
		if (mediaQuery.addEventListener) {
			mediaQuery.addEventListener("change", systemChange);
		} else if (mediaQuery.addListener) {
			mediaQuery.addListener(systemChange);
		}
	}
}());
