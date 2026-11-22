<script setup lang="ts">
import { CircleAlertIcon, Undo2Icon } from 'lucide-vue-next'
import { computed } from 'vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { type KeptChange, type Undo, propertyLabel } from '@/lib/history'
import type { ReviewKind } from '@/lib/reviewKinds'

// UndoAlert tells the person what the edit form undoes. A null undo did not
// load or is not a change.
const props = defineProps<{ kind: ReviewKind; undo: Undo | null }>()

// empty is true when no property of the version can go back.
const empty = computed(() => !!props.undo && Object.keys(props.undo.values).length === 0)

function keptText(k: KeptChange) {
  const who = k.editor?.name ?? 'Another person'
  return `${who} changed the ${propertyLabel(props.kind, k.property).toLowerCase()} again in version ${k.number}. The form keeps the current value.`
}
</script>

<template>
  <Alert v-if="undo">
    <Undo2Icon aria-hidden="true" />
    <AlertTitle>{{ empty ? `Nothing to undo in version ${undo.number}` : `Undo of version ${undo.number}` }}</AlertTitle>
    <AlertDescription>
      <p>{{ empty ? 'The form has the current values.' : 'Check the values, then save.' }}</p>
      <ul v-if="undo.kept.length" class="list-disc pl-5">
        <li v-for="k in undo.kept" :key="k.property">{{ keptText(k) }}</li>
      </ul>
    </AlertDescription>
  </Alert>
  <Alert v-else variant="destructive">
    <CircleAlertIcon aria-hidden="true" />
    <AlertTitle>This change cannot be undone here</AlertTitle>
    <AlertDescription>The version did not load, or it is not a change. The form has the current values.</AlertDescription>
  </Alert>
</template>
