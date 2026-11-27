# TagInput

Adds and removes tags on a restaurant or a recipe. Reviews have no tags. Project component, not part of shadcn-vue.

Lives in `@/components/ui/tag-input`. Props: `v-model` (string array), `max` (default 20), `suggest`, and `allowNew` (default true). Build it from `Badge`-like chips and an `Input`.

- Enter or comma adds the typed tag. Backspace in an empty field removes the last tag. Each chip has a remove button with `aria-label` "Remove <tag>".
- With `suggest`, the field is a WAI-ARIA combobox. Focus opens a `listbox` "Suggestions" with the most used tags. Typing changes them. See decision 00011.
- In a filter popover, `openOnFocus` is off. The popover focuses the field, and the list must not cover the other filters. Typing or ArrowDown opens it.
- A tag with the exact typed name is the first suggestion. "Add new tag" never shows for an existing tag.
- While new suggestions load, the arrow keys select nothing. Enter adds the stored form of the typed text.
- In a filter, a pick closes the list. In a form, the list shows the most used tags again.
- The accessible name of a suggestion is "<tag>, <count>". The comma is in a `sr-only` span. Do not add an `aria-label`.
- Each suggestion shows the name and a muted count, such as "4 restaurants". Arrow keys select a suggestion, Enter adds it, and Escape closes the list.
- `allowNew` adds the last option "Add new tag ‘text’" with a plus icon. The filter of a list page turns it off.
- The list is a `popover` surface under the field with `radius-md` and a shadow. Each option is at least 44 px high.
- It checks the API rules before the API does: 1 to 63 characters, the pattern of the `Tag` schema, unique, at most 20 tags.
- Tags have the lowercase letters and digits of all scripts, such as `café` or `ラーメン`. See decision 00011.
- The input changes free text to the stored form with `toTag`, such as `wood-fired` for `Wood Fired`. Show the change before the user adds the tag: the option "Add new tag ‘wood-fired’", the chip, and the live region "Added wood-fired".
- The field description tells the user that tags are saved in lowercase with hyphens.
- The one exception is Unicode normalization form C. The tag looks the same after the change. The API does the same change.
- Chips are `secondary` with `radius-sm`. The field border is `input`, `destructive` when invalid.
- Announce added and removed tags in a polite live region.
