<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import { useVModel } from "@vueuse/core"
import { cn } from "@/lib/utils"

const props = defineProps<{
 class?: HTMLAttributes["class"]
 defaultValue?: string | number
 modelValue?: string | number
}>()

const emits = defineEmits<{
 (e: "update:modelValue", payload: string | number): void
}>()

const modelValue = useVModel(props, "modelValue", emits, {
 passive: true,
 defaultValue: props.defaultValue,
})
</script>

<template>
 <textarea
 v-model="modelValue"
 data-slot="textarea"
 :class="cn('border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring aria-invalid:border-destructive flex field-sizing-content min-h-24 w-full rounded-lg border bg-card px-3.5 py-3 text-base leading-6 shadow-xs transition-[color,box-shadow] disabled:cursor-not-allowed disabled:opacity-50 md:text-[15px]', props.class)"
 />
</template>
