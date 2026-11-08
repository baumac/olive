<script setup lang="ts">
import type { User } from '@reviews/sdk'
import { LogOutIcon } from 'lucide-vue-next'
import { InitialsAvatar } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

defineProps<{
  user: User | null
  // signInPath is the sign in page with the current path as redirect.
  signInPath: string
  signInCurrent?: boolean
}>()

defineEmits<{ signOut: [] }>()
</script>

<template>
  <Button v-if="!user" variant="outline" as-child>
    <RouterLink :to="signInPath" :aria-current="signInCurrent ? 'page' : undefined">Sign in</RouterLink>
  </Button>
  <DropdownMenu v-else>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" class="rounded-full" :aria-label="`Account menu for ${user.name}`">
        <InitialsAvatar :name="user.name" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-56">
      <DropdownMenuLabel class="flex flex-col gap-0.5">
        <span class="truncate">{{ user.name }}</span>
        <span class="text-muted-foreground truncate text-xs font-normal">{{ user.email }}</span>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem @select="$emit('signOut')">
        <LogOutIcon aria-hidden="true" />
        Sign out
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
