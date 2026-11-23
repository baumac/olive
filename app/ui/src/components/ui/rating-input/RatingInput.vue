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
const baseId = props.id ?? fallbackId
const descriptionId = `${baseId}-description`
const endsId = `${baseId}-ends`
const hasEnds = computed(() => !!(props.lowest && props.highest))
const ariaDescribedBy = computed(
  () => [props.description && descriptionId, hasEnds.value && endsId, props.describedBy].filter(Boolean).join(' ') || undefined,
)
</script>

<template>
  <div class="grid gap-2">
    <p v-if="description" :id="descriptionId" class="text-muted-foreground text-sm">{{ description }}</p>
    <div class="grid w-full max-w-md gap-1">
      <RadioGroupRoot
        :id="props.id"
        v-model="value"
        orientation="horizontal"
        :aria-label="label"
        :aria-invalid="invalid ? 'true' : undefined"
        :aria-describedby="ariaDescribedBy"
        class="flex gap-0.5 sm:gap-1"
      >
        <RadioGroupItem
          v-for="n in max"
          :key="n"
          :value="String(n)"
          :aria-label="`${n} out of ${max}`"
          :class="
            cn(
              'border-input bg-card hover:bg-accent hover:text-accent-foreground inline-flex h-11 min-w-8 flex-1 items-center justify-center rounded-md border text-[15px] font-semibold tabular-nums',
              'data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground data-[state=checked]:border-primary',
              invalid && 'border-destructive',
            )
          "
        >
          {{ n }}
        </RadioGroupItem>
      </RadioGroupRoot>
      <!-- A screen reader gets the sentence, not the two words. -->
      <p v-if="hasEnds" :id="endsId" class="text-muted-foreground flex justify-between gap-4 text-xs">
        <span aria-hidden="true">{{ lowest }}</span>
        <span aria-hidden="true">{{ highest }}</span>
        <span class="sr-only">1 means {{ lowest }}. {{ max }} means {{ highest }}.</span>
      </p>
    </div>
  </div>
</template>
