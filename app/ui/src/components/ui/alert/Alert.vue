<script setup lang="ts">
import type { HTMLAttributes } from "vue"
import type { AlertVariants } from "."
import { cn } from "@/lib/utils"
import { alertVariants } from "."

const props = defineProps<{
  class?: HTMLAttributes["class"]
  variant?: AlertVariants["variant"]
}>()

// Errors interrupt a screen reader. Confirmations wait. Notes have no role.
const role = { destructive: "alert", success: "status", default: undefined }
</script>

<template>
  <div
    data-slot="alert"
    :class="cn(alertVariants({ variant }), props.class)"
    :role="role[variant ?? 'default']"
  >
    <slot />
  </div>
</template>
