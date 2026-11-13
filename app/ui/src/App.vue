<script setup lang="ts">
import { RouterView } from 'vue-router'
import 'vue-sonner/style.css'
import AppFooter from '@/components/app/AppFooter.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import { Toaster } from '@/components/ui/sonner'
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <AppHeader />
    <div class="flex-1">
      <RouterView :key="$route.path" />
    </div>
    <AppFooter />
  </div>
  <Toaster position="bottom-right" close-button :toast-options="{ duration: 4000, classes: { description: 'text-muted-foreground!' } }" class="z-[100]" />
</template>
