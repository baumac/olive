<script setup lang="ts">
import { RadioGroupItem, RadioGroupRoot } from 'reka-ui'
import { computed, useId } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  max: number
  label: string
  id?: string
  description?: string
  // lowest and highest tell what 1 and max mean.
  lowest?: string
  highest?: string
  invalid?: boolean
  describedBy?: string
}>()

const model = defineModel<number | null>({ default: null })

// Reka UI radio values are strings.
const value = computed({
  get: () => (model.value == null ? undefined : String(model.value)),
  set: (v) => (model.value = v == null ? null : Number(v)),
})

const fallbackId = useId()
const helpId = `${props.id ?? fallbackId}-description`
const hasHelp = computed(() => !!(props.description || (props.lowest && props.highest)))
const ariaDescribedBy = computed(() => [hasHelp.value && helpId, props.describedBy].filter(Boolean).join(' ') || undefined)
</script>

<template>
  <div class="grid gap-2">
    <RadioGroupRoot
      :id="props.id"
      v-model="value"
      orientation="horizontal"
      :aria-label="label"
      :aria-invalid="invalid ? 'true' : undefined"
      :aria-describedby="ariaDescribedBy"
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
    <div v-if="hasHelp" :id="helpId" class="text-muted-foreground grid gap-1 text-sm">
      <p v-if="description">{{ description }}</p>
      <p v-if="lowest && highest">1 means {{ lowest }}. {{ max }} means {{ highest }}.</p>
    </div>
  </div>
</template>
