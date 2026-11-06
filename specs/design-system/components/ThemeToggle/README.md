# ThemeToggle

Lets a person choose the light theme, the dark theme or the system setting. ADR 00002 requires it.

Project component built from shadcn-vue `DropdownMenu` with a `DropdownMenuRadioGroup`. Lives in `@/components/ui/theme-toggle`. No props. It reads and writes the `useColorMode` value from `@vueuse/core`, which stores the choice in `localStorage` and sets the `dark` class on `html`.

- Three options in this order: "Light", "Dark", "System". "System" is the default and follows `prefers-color-scheme`.
- The trigger is a ghost icon button in `AppHeader`. It shows the sun icon in light and the moon icon in dark. Its `aria-label` names the current choice, for example "Theme: system".
- The selected option shows a check and has `aria-checked="true"`.
- The inline script in `index.html` sets the class before first paint, so the page does not flash the wrong theme.
- Arrow keys move through options, Enter selects, Escape closes. Reka UI gives this behavior.
