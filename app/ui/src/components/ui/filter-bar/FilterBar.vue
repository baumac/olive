<script setup lang="ts">
import { SlidersHorizontalIcon, XIcon } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TagInput } from '@/components/ui/tag-input'

export interface Option {
  value: string
  label: string
}

export type FilterValue = string | string[] | undefined

export interface FilterDef {
  key: string
  label: string
  type: 'tags' | 'select'
  options?: Option[]
  help?: string
  // chip gives the chip text for one value, such as "Rating: 4 or more".
  chip: (value: string) => string
}

const props = defineProps<{
  defs: FilterDef[]
  sortOptions: Option[]
  sortLabel: string
  // status is the text of the result count, such as "12 reviews".
  status: string
}>()

const filters = defineModel<Record<string, FilterValue>>('filters', { required: true })
const sort = defineModel<string>('sort', { required: true })

// Reka UI Select needs a value for each item. "any" means no filter.
const any = 'any'
const tagError = ref('')

function set(key: string, value: FilterValue) {
  const next = { ...filters.value }
  if (value === undefined || value === any || (Array.isArray(value) && value.length === 0)) {
    delete next[key]
  } else {
    next[key] = value
  }
  filters.value = next
}

interface Chip {
  key: string
  value: string
  text: string
}

const chips = computed<Chip[]>(() =>
  props.defs.flatMap((def) => {
    const v = filters.value[def.key]
    const values = Array.isArray(v) ? v : v ? [v] : []
    return values.map((value) => ({ key: def.key, value, text: def.chip(value) }))
  }),
)

function removeChip(chip: Chip) {
  const v = filters.value[chip.key]
  set(chip.key, Array.isArray(v) ? v.filter((x) => x !== chip.value) : undefined)
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-wrap items-center gap-2">
      <Popover>
        <PopoverTrigger as-child>
          <Button variant="outline">
            <SlidersHorizontalIcon aria-hidden="true" />
            Filters
          </Button>
        </PopoverTrigger>
        <PopoverContent align="start" class="grid max-h-(--reka-popover-content-available-height) w-80 gap-4 overflow-y-auto">
          <div v-for="def in defs" :key="def.key" class="grid gap-2">
            <Label :for="`filter-${def.key}`">{{ def.label }}</Label>
            <template v-if="def.type === 'tags'">
              <TagInput
                :id="`filter-${def.key}`"
                :model-value="(filters[def.key] as string[] | undefined) ?? []"
                :invalid="!!tagError"
                :described-by="tagError ? `filter-${def.key}-message` : undefined"
                @update:model-value="set(def.key, $event)"
                @error="tagError = $event"
              />
              <p v-if="tagError" :id="`filter-${def.key}-message`" class="text-destructive text-sm">{{ tagError }}</p>
            </template>
            <Select
              v-else
              :model-value="(filters[def.key] as string | undefined) ?? any"
              @update:model-value="set(def.key, $event as string)"
            >
              <SelectTrigger :id="`filter-${def.key}`" class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="any">Any</SelectItem>
                <SelectItem v-for="o in def.options" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="def.help" class="text-muted-foreground text-sm">{{ def.help }}</p>
          </div>
        </PopoverContent>
      </Popover>
      <div class="flex items-center gap-2">
        <Label for="list-sort" class="text-muted-foreground font-normal">{{ sortLabel }}</Label>
        <Select v-model="sort">
          <SelectTrigger id="list-sort" class="min-w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="o in sortOptions" :key="o.value" :value="o.value">{{ o.label }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <p role="status" class="text-muted-foreground ml-auto text-sm">{{ status }}</p>
    </div>
    <ul v-if="chips.length" class="flex flex-wrap items-center gap-2" aria-label="Active filters">
      <li v-for="chip in chips" :key="`${chip.key}-${chip.value}`">
        <span class="border-input inline-flex items-center gap-1 rounded-full border py-0.5 pr-1 pl-3 text-xs font-medium">
          {{ chip.text }}
          <button
            type="button"
            class="hover:bg-accent hover:text-accent-foreground rounded-full p-0.5"
            :aria-label="`Remove filter ${chip.text.replace(':', '').toLowerCase()}`"
            @click="removeChip(chip)"
          >
            <XIcon class="size-3" aria-hidden="true" />
          </button>
        </span>
      </li>
      <li>
        <Button variant="link" size="sm" class="h-auto px-1" @click="filters = {}">Clear all</Button>
      </li>
    </ul>
  </div>
</template>
