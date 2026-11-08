<script setup lang="ts">
import { XIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { cn } from '@/lib/utils'
import { checkTag, maxTags } from '@/lib/validation'

const props = withDefaults(
  defineProps<{
    id?: string
    max?: number
    invalid?: boolean
    describedBy?: string
    placeholder?: string
  }>(),
  { max: maxTags, placeholder: 'italian' },
)

const tags = defineModel<string[]>({ default: () => [] })
const emit = defineEmits<{ error: [message: string] }>()

const text = ref('')
const announcement = ref('')

function add() {
  const tag = text.value.trim()
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
  announcement.value = `Added ${tag}`
}

function remove(tag: string) {
  tags.value = tags.value.filter((t) => t !== tag)
  announcement.value = `Removed ${tag}`
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' || e.key === ',') {
    e.preventDefault()
    add()
  } else if (e.key === 'Backspace' && text.value === '' && tags.value.length > 0) {
    remove(tags.value[tags.value.length - 1]!)
  }
}
</script>

<template>
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
      :placeholder="tags.length === 0 ? placeholder : ''"
      :aria-invalid="invalid ? 'true' : undefined"
      :aria-describedby="describedBy"
      :disabled="tags.length >= max"
      class="placeholder:text-muted-foreground min-w-24 flex-1 bg-transparent px-1 py-1 text-base outline-none focus-visible:outline-none disabled:cursor-not-allowed md:text-sm"
      @keydown="onKeydown"
      @input="emit('error', '')"
      @blur="add"
    />
    <span class="sr-only" aria-live="polite">{{ announcement }}</span>
  </div>
</template>
