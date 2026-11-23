<script setup lang="ts">
import { RouterView } from 'vue-router'
import 'vue-sonner/style.css'
import AppFooter from '@/components/app/AppFooter.vue'
import AppHeader from '@/components/app/AppHeader.vue'
import BottomNav from '@/components/app/BottomNav.vue'
import ApprovalBanner from '@/components/app/ApprovalBanner.vue'
import { Toaster } from '@/components/ui/sonner'
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
  <!-- Below md, the toasts stay above BottomNav. -->
  <Toaster position="bottom-right" :mobile-offset="{ bottom: 'calc(92px + env(safe-area-inset-bottom))' }" close-button :toast-options="{ duration: 4000, classes: { description: 'text-muted-foreground!' } }" class="z-[100]" />
</template>
