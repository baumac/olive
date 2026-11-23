<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { computed } from 'vue'
import { RouterView } from 'vue-router'
import 'vue-sonner/style.css'
import AppFooter from '@/components/app/AppFooter.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import BottomNav from '@/components/app/BottomNav.vue'
import ApprovalBanner from '@/components/app/ApprovalBanner.vue'
import { Toaster } from '@/components/ui/sonner'

// Below md, the toasts stay above BottomNav. The mobile offset of the
// toaster applies only up to 600px. It adds the safe area itself.
const md = useMediaQuery('(min-width: 768px)')
const toastOffset = computed(() => (md.value ? undefined : { bottom: 'calc(92px + env(safe-area-inset-bottom))' }))
</script>

<template>
  <!-- Below md, the bottom padding keeps the content above BottomNav. -->
  <div class="flex min-h-dvh flex-col pb-[calc(76px+env(safe-area-inset-bottom))] md:pb-0">
    <AppHeader />
    <ApprovalBanner />
    <div class="flex-1">
      <RouterView :key="$route.path" />
    </div>
    <AppFooter />
    <BottomNav />
  </div>
  <Toaster position="bottom-right" :offset="toastOffset" :mobile-offset="{ bottom: '92px' }" close-button :toast-options="{ duration: 4000, classes: { description: 'text-muted-foreground!' } }" class="z-[100]" />
</template>
