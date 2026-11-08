<script setup lang="ts">
import { RadioGroupItem, RadioGroupRoot } from 'reka-ui'
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  max: number
  label: string
  id?: string
  invalid?: boolean
  describedBy?: string
}>()

const model = defineModel<number | null>({ default: null })

// Reka UI radio values are strings.
const value = computed({
  get: () => (model.value == null ? undefined : String(model.value)),
  set: (v) => (model.value = v == null ? null : Number(v)),
})
</script>

<template>
  <RadioGroupRoot
    :id="props.id"
    v-model="value"
    orientation="horizontal"
    :aria-label="label"
    :aria-invalid="invalid ? 'true' : undefined"
    :aria-describedby="describedBy"
    class="flex flex-wrap gap-1"
  >
    <RadioGroupItem
      v-for="n in max"
      :key="n"
      :value="String(n)"
      :aria-label="`${n} out of ${max}`"
      :class="
        cn(
          'border-input bg-background hover:bg-accent hover:text-accent-foreground inline-flex size-8 items-center justify-center rounded-md border text-sm font-medium tabular-nums shadow-xs',
          'data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground data-[state=checked]:border-primary',
          invalid && 'border-destructive',
        )
      "
    >
      {{ n }}
    </RadioGroupItem>
  </RadioGroupRoot>
</template>
