<script setup lang="ts">
import { ThemeToggle } from '@/components/ui/theme-toggle'
import { currentUser } from '@/lib/session'
</script>

<template>
  <!-- A signed in person has the theme in the account menu. -->
  <footer v-if="!currentUser" class="mx-auto flex w-full max-w-6xl justify-end px-4 py-6 sm:px-6">
    <ThemeToggle />
  </footer>
</template>
