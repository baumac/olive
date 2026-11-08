<script setup lang="ts">
import { EyeIcon, EyeOffIcon } from 'lucide-vue-next'
import { ref } from 'vue'
import { Input } from '@/components/ui/input'

defineOptions({ inheritAttrs: false })

const model = defineModel<string>({ default: '' })
const visible = ref(false)
const input = ref<InstanceType<typeof Input> | null>(null)

defineExpose({ focus: () => (input.value?.$el as HTMLInputElement | undefined)?.focus() })
</script>

<template>
  <div class="relative">
    <Input ref="input" v-model="model" v-bind="$attrs" :type="visible ? 'text' : 'password'" class="pr-10" />
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground absolute inset-y-0 right-0 flex w-10 items-center justify-center rounded-md"
      aria-label="Show password"
      :aria-pressed="visible ? 'true' : 'false'"
      @click="visible = !visible"
    >
      <EyeOffIcon v-if="visible" class="size-4" aria-hidden="true" />
      <EyeIcon v-else class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>
