# Frontend style guide

Pow Hunter should feel like a clean snow report: quiet, flat, and precise. The numbers carry the page, so the chrome stays out of the way.

## Principles

- **Quiet by default.** Use white and slate surfaces with one accent color. The accent (glacier blue) is for primary actions and key data only.
- **Flat.** Separate things with 1px borders, not shadows. No gradients, glass effects, glows, or lift-on-hover. Shadows are only for floating layers (menus, dialogs).
- **Numbers are the hero.** Show snow amounts and day counts large, in tabular figures, like a weather readout (`12″`, `3 days`).
- **Less on the page.** One clear primary action per screen. Don't add filler cards, decorative icons, or "coming soon" content.
- **Honest.** Label example or illustrative data as such (see the "Example" chip on the home forecast card).

## Where things live

| File                                     | Purpose                                                                                           |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `src/theme.ts`                           | The single source of truth for colors, type, radii and MUI overrides. Also exports `tabularNums`. |
| `src/app.css`                            | A minimal reset plus `.sr-only`. Don't add component styles here.                                 |
| `src/components/header.tsx`              | The sticky top nav: wordmark, "Manage alerts" and "Create alert".                                 |
| `src/components/footer.tsx`              | A quiet footer.                                                                                   |
| `src/components/pageHeader.tsx`          | The page title (`h1`), optional subtitle, and an optional right-side action.                      |
| `src/components/sliderField.tsx`         | A slider with its label, hint and live value shown as a readout.                                  |
| `src/components/alertSettingsFields.tsx` | The minimum-snowfall and advance-notice sliders every alert carries.                              |
| `src/components/resortSelect.tsx`        | The checkbox resort picker that shows picks as chips.                                             |

## Tokens (defined in `theme.ts`)

- **Text:** `text.primary` is `#0b1220` and `text.secondary` is `#5b6474`.
- **Accent:** `primary.main` is `#1d6fe0`. Use `primary.light` (`#e8f1fd`) for tinted backgrounds such as chips and check badges.
- **Lines and surfaces:** `divider` is `#e6e9ef`. The page background is `#fbfcfd`, and Paper is white.
- **Type:** Inter (self-hosted via `@fontsource-variable/inter`). Headings are tight (negative letter-spacing, weight 600–700). Buttons don't use uppercase.
- **Radius:** 8px for inputs and buttons, 14px for outlined Paper and cards.

Use theme tokens (`'text.secondary'`, `'divider'`, `'primary.main'`). Don't hard-code hex values in components.

## Page patterns

- **Form or detail page:** `Container maxWidth="sm"` (or `"xs"` for sign in) with `py: { xs: 5, md: 8 }`, then `<PageHeader>`, then `<Paper variant="outlined">` holding the content.
- **List:** a single outlined `Paper` with rows separated by `borderTop: 1, borderColor: 'divider'`, rather than a stack of separate cards.
- **Empty state:** a centered outlined `Paper` with a muted icon, a short title, one line of copy, and one button.
- **Section label:** `Typography variant="overline"`, for example "3 ACTIVE" or "SNOW ALERTS BY TEXT".
- **Navigation:** the header handles it, so don't add "← Back to Home" buttons.

## Components

- **Buttons:** use `variant="contained"` for the one primary action, and `outlined` or text for everything else. Use `color="error"` only for destructive actions.
- **Numbers:** spread the shared style into `sx`, as in `sx={{ ...tabularNums, fontWeight: 600 }}`. Use the prime mark `″` for inches, not `"`.
- **Icons:** use small, outlined MUI icons (`DeleteOutline`, `Check`, `AcUnit`) in a muted or accent color. No solid colored icon tiles; a small check in a `primary.light` circle is fine for confirmations (see `routes/success.tsx`).
- **Sliders:** wrap them in a label row that shows the live value on the right (`components/sliderField.tsx`).
- **Copy:** short, plain, and sentence case. Talk about snow and resorts, not features.

## Accessibility and responsiveness

- Use one `h1` per page, and make heading levels follow the structure, not the visual size (use `variant` vs `component`).
- Icon-only buttons need an `aria-label`. Large visual numbers get readable text via `.sr-only` where needed (see `routes/manage.tsx`).
- Check the page at 390px wide with no horizontal scroll. Grids should collapse to one column (`size={{ xs: 12, sm: 6 }}`).

## Before you ship

- Run `bun test src/**/*.test.{ts,tsx}`, `bun x tsc --noEmit`, `bun run lint`, and `bun x prettier --write` on the files you changed.
- Look at the page in a browser at desktop and mobile widths, and compare it against the existing pages for spacing and type.
