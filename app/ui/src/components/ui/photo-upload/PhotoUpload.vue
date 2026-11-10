<script setup lang="ts">
import { ImagePlusIcon } from 'lucide-vue-next'
import { onBeforeUnmount, ref, useId } from 'vue'
import { Button } from '@/components/ui/button'
import { FormControl, FormDescription, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { checkAlt, checkPhotoFile, photoTypes } from '@/lib/validation'

const props = withDefaults(
  defineProps<{
    // actionLabel names the button that adds the selected file.
    actionLabel?: string
    busy?: boolean
    // error is the text of a failed upload.
    error?: string
  }>(),
  { actionLabel: 'Upload photo', error: '' },
)

const emit = defineEmits<{ add: [file: File, alt: string] }>()

const id = useId()
const input = ref<HTMLInputElement | null>(null)
const file = ref<File | null>(null)
const preview = ref('')
const alt = ref('')
const fileError = ref('')
const altError = ref('')

function release() {
  if (preview.value) {
    URL.revokeObjectURL(preview.value)
    preview.value = ''
  }
}

function select(event: Event) {
  const selected = (event.target as HTMLInputElement).files?.[0]
  release()
  file.value = null
  altError.value = ''
  if (!selected) {
    return
  }
  fileError.value = checkPhotoFile(selected)
  if (fileError.value) {
    // The same file can then cause a change event again.
    ;(event.target as HTMLInputElement).value = ''
    return
  }
  file.value = selected
  preview.value = URL.createObjectURL(selected)
}

function add() {
  altError.value = checkAlt(alt.value)
  if (file.value && !altError.value) {
    emit('add', file.value, alt.value)
  }
}

// reset clears the selection after the parent added the file.
function reset() {
  release()
  file.value = null
  alt.value = ''
  if (input.value) {
    input.value.value = ''
  }
}

onBeforeUnmount(release)
defineExpose({ reset })
</script>

<template>
  <div class="grid gap-3">
    <div class="grid gap-2">
      <label
        :for="`${id}-file`"
        class="border-input bg-background hover:bg-accent focus-within:outline-ring inline-flex h-9 w-fit cursor-pointer items-center gap-2 rounded-md border px-4 text-sm font-medium shadow-xs focus-within:outline-2 focus-within:outline-offset-2"
      >
        <ImagePlusIcon class="size-4" aria-hidden="true" />Add photo
        <input
          :id="`${id}-file`"
          ref="input"
          type="file"
          class="sr-only"
          :accept="photoTypes.join(',')"
          :aria-describedby="`${id}-hint${fileError || props.error ? ` ${id}-error` : ''}`"
          :aria-invalid="!!(fileError || props.error)"
          :disabled="busy"
          @change="select"
        >
      </label>
      <p :id="`${id}-hint`" class="text-muted-foreground text-sm">
        JPEG, PNG, or WebP of at most 10 MB. Add only photos of food, rooms, and menus.
      </p>
      <p v-if="fileError || props.error" :id="`${id}-error`" class="text-destructive text-sm">{{ fileError || props.error }}</p>
    </div>
    <div v-if="file" class="grid gap-3 rounded-md border p-3">
      <img :src="preview" alt="The selected photo" class="bg-muted max-h-48 w-fit rounded-md object-contain">
      <FormItem :id="`${id}-alt`" :error="altError" description>
        <FormLabel>Description</FormLabel>
        <FormControl>
          <Input v-model="alt" name="alt" maxlength="255" />
        </FormControl>
        <FormDescription>Optional. Describe the photo for people who cannot see it.</FormDescription>
        <FormMessage />
      </FormItem>
      <div class="flex flex-wrap gap-2">
        <Button type="button" :disabled="busy" @click="add">{{ busy ? 'Uploading…' : actionLabel }}</Button>
        <Button type="button" variant="outline" :disabled="busy" @click="reset">Cancel</Button>
      </div>
    </div>
  </div>
</template>
