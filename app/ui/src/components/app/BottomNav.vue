<script setup lang="ts">
import { ChefHatIcon, HouseIcon, PlusIcon, UtensilsIcon } from 'lucide-vue-next'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { pantryIcon } from '@/lib/pantry'
import { isCurrentSection } from '@/lib/sections'
import { currentUser } from '@/lib/session'

// BottomNav has the sections below md. The journal is in UserMenu. Five
// slots then fit at 375px.
const route = useRoute()

const items = computed(() => [
  { path: '/', label: 'Table', icon: HouseIcon },
  { path: '/restaurants', label: 'Restaurants', icon: UtensilsIcon },
  ...(currentUser.value ? [{ path: '/reviews/new', label: 'Add review', icon: PlusIcon, add: true }] : []),
  { path: '/recipes', label: 'Recipes', icon: ChefHatIcon },
  { path: '/pantry', label: 'Pantry', icon: pantryIcon },
])
</script>

<template>
  <nav
    aria-label="Sections"
    class="bg-card fixed inset-x-0 bottom-0 z-40 flex h-[calc(76px+env(safe-area-inset-bottom))] items-stretch border-t px-1 pb-[calc(6px+env(safe-area-inset-bottom))] md:hidden"
  >
    <RouterLink
      v-for="item in items"
      :key="item.path"
      :to="item.path"
      :aria-current="isCurrentSection(item.path, route.path) ? 'page' : undefined"
      :class="[
        'flex flex-1 flex-col items-center gap-1 text-[11px] leading-[14px] font-semibold',
        'add' in item ? 'text-foreground justify-end pb-2' : 'justify-center',
        !('add' in item) && (isCurrentSection(item.path, route.path) ? 'text-primary' : 'text-muted-foreground'),
      ]"
    >
      <span
        v-if="'add' in item"
        class="bg-primary text-primary-foreground flex size-13 items-center justify-center rounded-full shadow-[0_2px_5px_0_#00000052]"
      >
        <component :is="item.icon" class="size-6" :stroke-width="2.4" aria-hidden="true" />
      </span>
      <component :is="item.icon" v-else class="size-[22px]" aria-hidden="true" />
      <span>{{ item.label }}</span>
    </RouterLink>
  </nav>
</template>
