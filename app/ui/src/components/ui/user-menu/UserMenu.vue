<script setup lang="ts">
import type { User } from '@reviews/sdk'
import { ListIcon, LogOutIcon, MessageSquareIcon } from 'lucide-vue-next'
import { initials } from '@/lib/format'
import { InitialsAvatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { themeOptions, useTheme } from '@/composables/useTheme'

defineProps<{
  user: User | null
  // signInPath is the sign in page with the current path as redirect.
  signInPath: string
  signInCurrent?: boolean
  myReviewsPath: string
  // myListsPath is the page with the lists of the user.
  myListsPath?: string
}>()

defineEmits<{ signOut: [] }>()

const { choice } = useTheme()
</script>

<template>
  <Button v-if="!user" variant="outline" as-child>
    <RouterLink :to="signInPath" :aria-current="signInCurrent ? 'page' : undefined">Sign in</RouterLink>
  </Button>
  <DropdownMenu v-else>
    <DropdownMenuTrigger as-child>
      <!-- The name starts with the visible initials. Speech input can then use them. -->
      <Button variant="ghost" size="icon" class="rounded-full" :aria-label="`${initials(user.name)}, account menu for ${user.name}`">
        <InitialsAvatar :name="user.name" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-56">
      <DropdownMenuLabel class="flex flex-col gap-0.5">
        <span class="truncate">{{ user.name }}</span>
        <span class="text-muted-foreground truncate text-xs font-normal">{{ user.email }}</span>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem as-child>
        <RouterLink :to="myReviewsPath">
          <MessageSquareIcon aria-hidden="true" />
          My reviews
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuItem v-if="myListsPath" as-child>
        <RouterLink :to="myListsPath">
          <ListIcon aria-hidden="true" />
          My lists
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuLabel id="user-menu-theme" class="text-muted-foreground text-xs font-normal">Theme</DropdownMenuLabel>
      <DropdownMenuRadioGroup v-model="choice" aria-labelledby="user-menu-theme">
        <DropdownMenuRadioItem v-for="o in themeOptions" :key="o.value" :value="o.value">{{ o.label }}</DropdownMenuRadioItem>
      </DropdownMenuRadioGroup>
      <DropdownMenuSeparator />
      <DropdownMenuItem @select="$emit('signOut')">
        <LogOutIcon aria-hidden="true" />
        Sign out
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
