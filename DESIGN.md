# Gibloux Volley Design System

This document is the implementation contract for the static Gibloux Volley pages: upcoming matches, played matches, and rankings. It records the existing light interface as the baseline and defines the compatible dark theme.

## 1 Principles

1. Keep the pages calm, compact, and data-first. Match and ranking information takes priority over decoration.
2. Preserve the current light appearance: white page background, `#333333` body text, `#545454` navigation, Helvetica-family typography, and the existing Gibloux Volley SVG logo.
3. Use the club mark as the source of truth for brand color. Gibloux blue is `#417BA6`; Gibloux yellow is `#FDC500`.
4. Add theme support by mapping semantic tokens, never by scattering theme-specific color values through component rules.
5. Maintain the current static HTML, jQuery, and DataTables model. New UI work must fit that model unless a separate decision changes it.
6. Prefer clear labels and visible controls. Icons support meaning; they never replace a required text label.

## 2 Foundations

### Brand identity

The primary identity is the existing `static/img/logo_giblouxvolley.svg` and its SVG favicon. Keep the logo's native blue and yellow fills. Do not recolor, redraw, crop, or substitute the logo for a text treatment.

The pages use a centered `fw-container`, currently capped at `1250px`, with `1rem` horizontal padding and a broad data area. The content is intentionally simple: page menu, page title, team filter, then DataTables content.

### Theme contract

Theme state belongs on the root element: `html[data-theme="light"]` or `html[data-theme="dark"]`. A manual choice is stored under the `gibloux-theme` localStorage key as `light` or `dark`.

If no stored choice exists, use `prefers-color-scheme` as the default. Apply the resolved `data-theme` in a small inline script in the document head before stylesheets render, so the page does not flash the wrong palette. The toggle changes the root attribute, persists the selected value, and exposes its state with `aria-pressed` and a French accessible name describing the action.

All new theme work must live in one consolidated token block near the top of the stylesheet. Component rules consume only those tokens. Do not add isolated `html.dark` overrides.

## 3 Color

### Semantic tokens

Use these exact semantic CSS custom properties. The light values preserve the current appearance. The dark values are the approved dark palette.

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `--color-page` | `#FFFFFF` | `#101820` | Page canvas |
| `--color-surface` | `#FFFFFF` | `#162230` | Controls and table surface |
| `--color-surface-elevated` | `#F7F9FA` | `#1E2D3B` | Raised or responsive detail surface |
| `--color-text` | `#333333` | `#F2F7FA` | Primary text and headings |
| `--color-text-muted` | `#545454` | `#B8C6D1` | Navigation, labels, secondary information |
| `--color-border` | `#E6E6E6` | `#31475B` | Table rules and control borders |
| `--color-link` | `#417BA6` | `#8BC7F0` | Interactive text links |
| `--color-link-hover` | `#2F6188` | `#B9E2FA` | Hovered interactive text |
| `--color-brand-blue` | `#417BA6` | `#417BA6` | Logo-adjacent brand accent only |
| `--color-brand-yellow` | `#FDC500` | `#FFD84D` | Club and cup accent |
| `--color-on-brand-yellow` | `#333333` | `#101820` | Text or icon over yellow |
| `--color-row-hover` | `#F6F6F6` | `#223548` | Hovered table row |
| `--color-row-selected` | `#E8F1F7` | `#203B50` | Selected ranking row |
| `--color-focus` | `#417BA6` | `#8BC7F0` | Keyboard focus ring |
| `--color-icon` | `#545454` | `#B8C6D1` | Non-logo SVG icons |

Use a token block with a light default and a dark override. The media query supplies the default only when no manual root attribute is present; explicit root attributes always win.

```css
:root {
  --color-page: #FFFFFF;
  --color-surface: #FFFFFF;
  --color-surface-elevated: #F7F9FA;
  --color-text: #333333;
  --color-text-muted: #545454;
  --color-border: #E6E6E6;
  --color-link: #417BA6;
  --color-link-hover: #2F6188;
  --color-brand-blue: #417BA6;
  --color-brand-yellow: #FDC500;
  --color-on-brand-yellow: #333333;
  --color-row-hover: #F6F6F6;
  --color-row-selected: #E8F1F7;
  --color-focus: #417BA6;
  --color-icon: #545454;
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) {
    --color-page: #101820;
    --color-surface: #162230;
    --color-surface-elevated: #1E2D3B;
    --color-text: #F2F7FA;
    --color-text-muted: #B8C6D1;
    --color-border: #31475B;
    --color-link: #8BC7F0;
    --color-link-hover: #B9E2FA;
    --color-brand-blue: #417BA6;
    --color-brand-yellow: #FFD84D;
    --color-on-brand-yellow: #101820;
    --color-row-hover: #223548;
    --color-row-selected: #203B50;
    --color-focus: #8BC7F0;
    --color-icon: #B8C6D1;
  }
}

html[data-theme="dark"] {
  --color-page: #101820;
  --color-surface: #162230;
  --color-surface-elevated: #1E2D3B;
  --color-text: #F2F7FA;
  --color-text-muted: #B8C6D1;
  --color-border: #31475B;
  --color-link: #8BC7F0;
  --color-link-hover: #B9E2FA;
  --color-brand-blue: #417BA6;
  --color-brand-yellow: #FFD84D;
  --color-on-brand-yellow: #101820;
  --color-row-hover: #223548;
  --color-row-selected: #203B50;
  --color-focus: #8BC7F0;
  --color-icon: #B8C6D1;
}
```

## 4 Typography/Spacing

Use the existing stack: `"Helvetica Neue", HelveticaNeue, Helvetica, Arial, sans-serif`. The base text remains `90%` with a `1.45em` line height. Do not introduce a web font for this interface.

| Role | Size | Weight | Notes |
| --- | --- | --- | --- |
| Body and table cells | Inherit, current 90% base | 400 | Primary match and ranking data |
| Page menu | `medium` | 400 | Uppercase, `#545454` in light mode |
| Page title | `1.5em` | 700 | Single-line ellipsis is retained where width requires it |
| Filter label | `medium` | 400 | Keep adjacent to its select control |
| Cup label | Visually hidden | Inherited | Remains available to assistive technology |

Use the existing spatial rhythm: `1rem` container side padding, `3rem` bottom container padding, `20px` above the page title, `30px` above the team filter, and `10px` between page-menu links. Add new spacing only from a four-point scale: `4px`, `8px`, `16px`, `24px`, `32px`.

## 5 Components

### Page Shell

The shell is the `html`, `body`, `.fw-container`, `.fw-body`, and `.content` hierarchy. It owns `--color-page`, `--color-text`, the Helvetica stack, the centered maximum width, and content padding. It must cover the full viewport and never use a decorative background that competes with the data.

### Page Menu

The page menu contains the logo link and the three uppercase page links. Links use `--color-text-muted`, have no underline by default, and retain the existing modest opacity hover treatment. The active page is visibly subdued, matching the current `.active` behavior, while still meeting contrast requirements. Links must remain keyboard reachable and show the shared focus ring.

### Logo

`#logo` is the existing SVG background treatment, sized at `254px` by `138px`. It links to the club site. Keep the natural SVG palette in both themes, provide an accessible name for the destination, and preserve a visible focus indicator around the link. Do not use the logo as a theme toggle or as the only navigation affordance.

### Theme Toggle

The toggle is a real `button`, placed in the page-menu area without displacing the existing page links. It has a minimum `44px` by `44px` target, uses an SVG sun or moon icon plus an accessible French label, and sets `aria-pressed` to reflect whether dark mode is active. Its visible state uses `--color-surface`, `--color-border`, `--color-icon`, and `--color-focus`.

It works with mouse, touch, Enter, and Space. It does not require a confirmation dialog, and it must not rely on color alone to describe the current theme.

### Team Filter

The team filter is a visible `label` paired with `#sel_team_id`. Keep the current label and select adjacent, preserving the `30px` top spacing. The select uses `--color-surface`, `--color-text`, and `--color-border`; its focus state uses the shared focus ring. Changing the value refreshes the relevant DataTable and, for upcoming matches, conditionally exposes the calendar export link.

### DataTable

DataTables remains the data presentation layer. Table text uses `--color-text`; headers, borders, pagination, filters, and information controls must use the semantic tokens. Rows use `--color-surface`, hover uses `--color-row-hover`, and the table must retain readable headers, sorting affordances, and the current French DataTables locale.

Sorting controls must remain operable by keyboard. Do not hide table data to solve narrow layouts. The ranking table preserves its current priority order: rank, team, and points remain visible before secondary statistics.

### Responsive Child Row

When DataTables collapses columns, its child row presents omitted values as labeled details. Use `--color-surface-elevated`, `--color-text`, and `--color-border`. The control cell remains visibly interactive, supports keyboard activation, exposes its expanded state with `aria-expanded`, and updates the row's child content without losing focus.

### Selected Row

The ranking row for the currently selected team uses `--color-row-selected`, not a change to text alone. It retains readable text and does not depend only on color: the selected state should remain available through DataTables selection semantics and an accessible row state where supported.

### Cup Badge

Cup matches show the existing Feather `award` SVG beside the league label. The icon inherits `--color-brand-yellow` or `--color-icon` only when contrast remains adequate. Keep the visually hidden `Coupe` text for screen readers. Do not replace the SVG with an emoji or a text glyph.

## 6 Motion/Interaction

The interface is mostly static. Retain the current brief opacity response on link hover, but do not add decorative motion. Theme changes may transition only `color`, `background-color`, and `border-color` for up to `150ms`, and must honor `prefers-reduced-motion: reduce` by disabling nonessential transitions.

Every interactive element uses `:focus-visible` with a `2px` solid `--color-focus` outline and a `2px` offset. Do not remove outlines on focus. Hover styles supplement focus styles; they never replace them.

## 7 Responsive/Accessibility

### Responsive rules

Validate each page at these widths:

| Viewport | Required behavior |
| --- | --- |
| `375px` | Menu and toggle stay usable without horizontal page overflow. DataTables collapses secondary columns into child rows. Ranking cells retain the existing compact padding and wrapping behavior. |
| `768px` | Container padding and menu remain balanced. Tables show the maximum practical columns while child rows hold the remainder. |
| `1280px` | The centered shell reaches its intended broad desktop presentation. Full table data, menu, logo, filter, and theme toggle align without crowding. |

The logo may wrap independently from navigation only when needed. Do not make the data table unreadably small or conceal meaningful columns at any viewport.

### Accessibility constraints

1. Keep `lang="fr"` and French labels, titles, and DataTables locale strings.
2. Use native links, buttons, labels, selects, and table semantics before adding ARIA. ARIA describes dynamic state, including the theme toggle and responsive row expansion, but does not replace native behavior.
3. Every interactive control must be reachable in a logical Tab order and operable by keyboard.
4. Maintain WCAG AA contrast for normal text, controls, focus rings, and selected states in both themes.
5. Use SVG icons with accessible names where an icon conveys an action. Decorative SVGs, including the cup award beside its hidden text label, use `aria-hidden="true"`. Never use emojis as icons.
6. Preserve visible text alternatives for dynamically revealed details and ensure DataTables redraws do not silently discard keyboard focus.

## 8 Accepted Debt

`static/css/main.css` is a legacy, third-party-derived stylesheet containing broad resets, unrelated DataTables demonstration styles, duplicated rules, deprecated syntax, and existing `html.dark` rules from its source. It remains accepted debt because replacing or reorganizing it is outside the static UI theme scope.

New theme work must not extend that debt. Place the consolidated semantic token block near the top of the maintained stylesheet, map all new component colors to it, and use root `data-theme` selectors rather than adding more `html.dark` overrides. A later dedicated stylesheet cleanup may remove legacy rules after visual regression checks at `375px`, `768px`, and `1280px`.
