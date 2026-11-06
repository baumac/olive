# Tabs

Switches between views of the same place. On the create review screen it selects the review kind.

From shadcn-vue (`@/components/ui/tabs`, built on Reka UI `Tabs`). Parts: `TabsList`, `TabsTrigger`, `TabsContent`. The consumer gives a `v-model` and one trigger and panel per value.

- The tab values are the API `kind` values: `basic` ("Basic") and `joc_de_cartes` ("Joc de cartes").
- The `basic` panel holds one `RatingInput` with `max` 5. The `joc_de_cartes` panel holds four with `max` 10: space, service, food, price.
- On the edit screen, the kind cannot change. The API refuses a different `kind` with `422`. Show the kind as text, not as tabs.
- The list track is `muted`. The active tab is `background` with `shadow-sm`.
- Inactive labels are `muted-foreground` on `muted`: 4.8:1 light, 6.1:1 dark.
