# MapView

A Leaflet map with the tiles of OpenStreetMap and the restaurants as score pins. Project component.

Lives in `@/components/ui/map`. Props: `config` (from `/map-config.json`), `label` (the name for screen readers), `center`, `zoom`, `markers`, `cluster`, `clusterLabel`. Decision 00008 has the behavior.

- Each pin is the score stamp of a restaurant: a 44px disc in `map-pin` with a 2px `map-pin-foreground` ring and the overall in the serif. A restaurant without reviews has an empty stamp. Its name ends with "No reviews yet".
- The selected pin is 52px, in `map-pin-foreground` with the number in `map-pin`. It has `aria-pressed="true"` and sits above the others.
- Pins less than 44px apart join into a 46px cluster pin in `map-cluster` with a 3px `map-cluster-foreground` ring and the count. A click zooms until its pins are apart.
- The preferred location is a 16px `map-location` dot with a white ring and a halo. It is not a button.
- Each pin is a button named with the name and the overall, such as "Luigi's Trattoria, 7.8". Enter and Space select it.
- The `map-` colors are fixed in both themes, because the tiles are light in both. Do not use theme tokens on the map.
- The zoom buttons are 44px. The credit "© OpenStreetMap" is at the bottom right.
- The map has `isolation: isolate`, so the Leaflet layers stay under `AppHeader`, `BottomNav` and dialogs.
- The list view is the text alternative of the map. From `lg` the restaurants page shows both side by side. The list keeps its pages. A pin of a restaurant that the list has not loaded shows its card above the list.
