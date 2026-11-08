<script setup lang="ts">
import { RouterView } from 'vue-router'
import 'vue-sonner/style.css'
import AppHeader from '@/components/app/AppHeader.vue'
import { Toaster } from '@/components/ui/sonner'
</script>

<template>
  <AppHeader />
  <RouterView :key="$route.path" />
  <Toaster position="bottom-right" close-button :toast-options="{ duration: 4000, classes: { description: 'text-muted-foreground!' } }" class="z-[100]" />
</template>
