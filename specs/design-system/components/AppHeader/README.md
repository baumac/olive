# AppHeader

The bar at the top of every screen: the logo and the account menu, and from `md` the sections and "Add review".

Project component. Lives in `@/components/app/AppHeader.vue`, outside `components/ui/`, because it knows the routes. No props; it reads the route and the signed in user.

- Below `md`: the olive mark alone at 30px, linking to the table, and `UserMenu` at the right end. The sections are in `BottomNav`. The height is `header-height` (60px).
- From `md`: the logo lockup at 36px, a `nav` with the name "Sections" and the links "Table", "Restaurants" and "Recipes", then "Add review" as the default `Button` and `UserMenu`. The height is `header-height-md` (72px). See decision 00009.
- Use the light or dark logo file from the Logos group to match the theme. The link's accessible name is "Sponga47 Reviews, home". It opens `/`, the table.
- A section link is current on each page of its section, such as `/recipes/{id}/reviews/new`. The current link has `aria-current="page"` and the `accent` fill with `accent-foreground` text. The others are `muted-foreground` and turn `foreground` on hover.
- The section links are pills at least 44px high, with 14px side padding.
- A signed out visitor has no "Add review". `UserMenu` shows "Sign in". The theme is in the page footer for a signed out person, and in `UserMenu` for a signed in person.
- The content is at most `content-max` wide, with 16px side padding below `sm` and 24px from `sm`. It has a `border` hairline at the bottom.
- The header is sticky (`sticky top-0`) with `z-header`. Its fill is `background`, so content does not show through.
- The first item in the page is a "Skip to content" link that shows on focus.
