<script setup lang="ts">
import { PlusIcon, XIcon } from 'lucide-vue-next'
import { computed, onBeforeUnmount, ref, useId } from 'vue'
import { cn } from '@/lib/utils'
import { checkTag, maxTags } from '@/lib/validation'

export interface TagSuggestion {
  name: string
  // label tells how often the tag is used, such as "4 restaurants".
  label: string
}

const props = withDefaults(
  defineProps<{
    id?: string
    max?: number
    invalid?: boolean
    describedBy?: string
    placeholder?: string
    // suggest gets the existing tags that match the text. An empty text
    // gets the most used tags.
    suggest?: (text: string) => Promise<TagSuggestion[]>
    // allowNew adds the option "Add new tag" for a text without a match.
    allowNew?: boolean
    // openOnFocus shows the most used tags on focus. Without it, typing or
    // ArrowDown opens the suggestions.
    openOnFocus?: boolean
  }>(),
  { max: maxTags, placeholder: 'italian', allowNew: true, openOnFocus: true, suggest: undefined, id: undefined, describedBy: undefined },
)

const tags = defineModel<string[]>({ default: () => [] })
const emit = defineEmits<{ error: [message: string] }>()

const text = ref('')
const announcement = ref('')
const open = ref(false)
const suggestions = ref<TagSuggestion[]>([])
const active = ref(-1)
const listboxId = `${props.id ?? useId()}-suggestions`

interface Option {
  name: string
  label: string
  isNew: boolean
}

const options = computed<Option[]>(() => {
  const typed = text.value.trim().normalize('NFC')
  const found = suggestions.value.filter((s) => !tags.value.includes(s.name)).map((s) => ({ ...s, isNew: false }))
  if (props.allowNew && typed !== '' && !tags.value.includes(typed) && !found.some((s) => s.name === typed)) {
    found.push({ name: typed, label: '', isNew: true })
  }
  return found
})

const expanded = computed(() => open.value && options.value.length > 0 && tags.value.length < props.max)

function optionId(index: number) {
  return `${listboxId}-${index}`
}

// The suggestions wait for a pause in the typing. A late answer of an older
// text does not replace a newer one. While an answer is pending, the arrow
// keys select nothing. The options then can be of the old text.
const debounce = 200
let timer: ReturnType<typeof setTimeout> | undefined
let generation = 0
const pending = ref(false)

function loadSuggestions() {
  if (!props.suggest) {
    return
  }
  clearTimeout(timer)
  const g = ++generation
  pending.value = true
  const typed = text.value.trim()
  timer = setTimeout(async () => {
    let found: TagSuggestion[] = []
    try {
      found = await props.suggest!(typed)
    } catch {
      // Without suggestions, the typed text still works.
    }
    if (g !== generation) {
      return
    }
    suggestions.value = found
    pending.value = false
  }, debounce)
}

onBeforeUnmount(() => clearTimeout(timer))

function add(value = text.value) {
  const tag = value.trim().normalize('NFC')
  if (tag === '') {
    return
  }
  const error = checkTag(tag, tags.value.slice(0, props.max))
  emit('error', error)
  if (error) {
    return
  }
  tags.value = [...tags.value, tag]
  text.value = ''
  active.value = -1
  announcement.value = `Added ${tag}`
  // A filter closes the list, so it does not cover the other filters.
  if (props.openOnFocus) {
    loadSuggestions()
  } else {
    open.value = false
  }
}

function remove(tag: string) {
  tags.value = tags.value.filter((t) => t !== tag)
  announcement.value = `Removed ${tag}`
}

function move(step: number) {
  if (!open.value) {
    open.value = true
    loadSuggestions()
  }
  const n = options.value.length
  if (n === 0 || pending.value) {
    return
  }
  active.value = active.value < 0 ? (step > 0 ? 0 : n - 1) : (active.value + step + n) % n
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    move(e.key === 'ArrowDown' ? 1 : -1)
  } else if (e.key === 'Escape' && expanded.value) {
    e.preventDefault()
    open.value = false
    active.value = -1
  } else if (e.key === 'Enter' || e.key === ',') {
    e.preventDefault()
    const option = expanded.value && e.key === 'Enter' ? options.value[active.value] : undefined
    add(option?.name)
  } else if (e.key === 'Backspace' && text.value === '' && tags.value.length > 0) {
    remove(tags.value[tags.value.length - 1]!)
  }
}

function onInput() {
  emit('error', '')
  open.value = true
  active.value = -1
  loadSuggestions()
}

function onFocus() {
  if (props.openOnFocus) {
    open.value = true
    loadSuggestions()
  }
}

function onBlur() {
  open.value = false
  active.value = -1
  add()
}
</script>

<template>
  <div class="relative">
    <div
      :class="
        cn(
          'border-input dark:bg-input/30 flex min-h-9 w-full flex-wrap items-center gap-1.5 rounded-md border px-2 py-1 shadow-xs',
          'focus-within:border-ring focus-within:outline-ring focus-within:outline-2 focus-within:outline-offset-2',
          invalid && 'border-destructive',
        )
      "
    >
      <span
        v-for="tag in tags"
        :key="tag"
        class="bg-secondary text-secondary-foreground inline-flex items-center gap-1 rounded-sm py-0.5 pr-1 pl-2 text-xs font-medium"
      >
        {{ tag }}
        <button
          type="button"
          :aria-label="`Remove ${tag}`"
          class="hover:bg-accent hover:text-accent-foreground rounded-sm p-0.5"
          @click="remove(tag)"
        >
          <XIcon class="size-3" aria-hidden="true" />
        </button>
      </span>
      <input
        :id="id"
        v-model="text"
        type="text"
        role="combobox"
        autocomplete="off"
        aria-autocomplete="list"
        :aria-expanded="expanded ? 'true' : 'false'"
        :aria-controls="listboxId"
        :aria-activedescendant="expanded && active >= 0 ? optionId(active) : undefined"
        :placeholder="tags.length === 0 ? placeholder : ''"
        :aria-invalid="invalid ? 'true' : undefined"
        :aria-describedby="describedBy"
        :disabled="tags.length >= max"
        class="placeholder:text-muted-foreground min-w-24 flex-1 bg-transparent px-1 py-1 text-base outline-none focus-visible:outline-none disabled:cursor-not-allowed md:text-sm"
        @keydown="onKeydown"
        @input="onInput"
        @focus="onFocus"
        @blur="onBlur"
      />
    </div>
    <ul
      v-show="expanded"
      :id="listboxId"
      role="listbox"
      aria-label="Suggestions"
      class="bg-popover text-popover-foreground absolute z-50 mt-1 max-h-60 w-full overflow-y-auto rounded-md border p-1 shadow-md"
    >
      <li
        v-for="(option, i) in options"
        :id="optionId(i)"
        :key="(option.isNew ? 'new:' : '') + option.name"
        role="option"
        :aria-selected="i === active ? 'true' : 'false'"
        :class="cn('flex min-h-11 cursor-pointer items-center gap-2 rounded-sm px-2 text-sm', i === active && 'bg-accent text-accent-foreground')"
        @mousedown.prevent="add(option.name)"
        @mousemove="active = i"
      >
        <template v-if="option.isNew">
          <PlusIcon class="size-4 shrink-0" aria-hidden="true" />
          <span class="wrap-anywhere">Add new tag ‘{{ option.name }}’</span>
        </template>
        <template v-else>
          <span class="flex-1 wrap-anywhere">{{ option.name }}</span>
          <span class="text-muted-foreground shrink-0 text-xs"><span class="sr-only">, </span>{{ option.label }}</span>
        </template>
      </li>
    </ul>
    <span class="sr-only" aria-live="polite">{{ announcement }}</span>
  </div>
</template>
