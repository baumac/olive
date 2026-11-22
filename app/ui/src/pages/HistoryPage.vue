<script setup lang="ts">
import { ArrowLeftIcon, HistoryIcon, Undo2Icon } from 'lucide-vue-next'
import { computed, watch } from 'vue'
import DateText from '@/components/app/DateText.vue'
import ErrorPage from '@/components/app/ErrorPage.vue'
import PagedList from '@/components/app/PagedList.vue'
import ReviewerLink from '@/components/app/ReviewerLink.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { useLoad } from '@/composables/useLoad'
import { usePagedList } from '@/composables/usePagedList'
import { usePageTitle } from '@/composables/usePageTitle'
import { type AnyVersion, changedProperties, propertyLabel, valueText } from '@/lib/history'
import { type ReviewKind, kinds } from '@/lib/reviewKinds'

// HistoryPage lists the versions of a restaurant or a recipe. See decision
// 00006.
const props = defineProps<{ kind: ReviewKind; parentId: string }>()

const info = computed(() => kinds[props.kind])
const parentPath = computed(() => `${info.value.path}/${props.parentId}`)
const parent = useLoad(() => info.value.getParent(props.parentId))
const versions = usePagedList<AnyVersion>((cursor) => info.value.listVersions(props.parentId, { cursor }))

watch(
  () => props.parentId,
  () => {
    void parent.load()
    void versions.reload()
  },
  { immediate: true },
)

usePageTitle(() => (parent.data.value ? `History of ${parent.data.value.name}` : 'History'))

function changes(version: AnyVersion) {
  const before = version.before as Record<string, unknown>
  const after = version.after as Record<string, unknown>
  return changedProperties(props.kind, version).map((property) => ({
    property,
    label: propertyLabel(props.kind, property),
    before: valueText(property, before[property]),
    after: valueText(property, after[property]),
  }))
}
</script>

<template>
  <ErrorPage v-if="parent.failed.value" :status="parent.failed.value.status" @retry="parent.load" />
  <main v-else id="main" class="mx-auto flex max-w-3xl flex-col gap-6 px-4 py-8 sm:px-6">
    <Button variant="ghost" size="sm" class="self-start" as-child>
      <RouterLink :to="parentPath"><ArrowLeftIcon aria-hidden="true" />Back to the {{ info.noun }}</RouterLink>
    </Button>
    <Skeleton v-if="parent.loading.value" class="h-10 w-2/3" />
    <h1 v-else class="text-4xl font-extrabold tracking-tight break-words">History of {{ parent.data.value?.name }}</h1>
    <p class="text-muted-foreground">
      Each change makes a new version. To fix a wrong change, undo it and save. The undo is a new version.
    </p>
    <PagedList
      :list="versions"
      noun="versions"
      skeleton-class="h-40 rounded-xl"
      :skeleton-count="3"
      unfiltered
    >
      <template #item="{ item }">
        <Card>
          <CardHeader>
            <CardTitle as="h2">Version {{ item.number }}</CardTitle>
            <CardDescription>
              <ReviewerLink v-if="item.editor" :creator="item.editor" />
              <template v-else-if="item.source === 'admin'">An admin</template>
              <template v-else>Deleted user</template>
              ·
              <DateText :created="item.created" />
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p v-if="item.action === 'create'">Added the {{ info.noun }}.</p>
            <dl v-else class="grid gap-4">
              <div v-for="c in changes(item)" :key="c.property" class="grid gap-1">
                <dt class="font-medium">{{ c.label }}</dt>
                <dd v-for="side in (['before', 'after'] as const)" :key="side" class="grid grid-cols-[4rem_1fr] gap-2">
                  <span class="text-muted-foreground text-sm">{{ side === 'before' ? 'Before' : 'After' }}</span>
                  <span v-if="c[side] === null" class="text-muted-foreground italic">None</span>
                  <ol v-else-if="Array.isArray(c[side])" class="list-decimal pl-5">
                    <li v-for="(line, i) in c[side]" :key="i" class="whitespace-pre-line">{{ line }}</li>
                  </ol>
                  <span v-else class="break-words whitespace-pre-line">{{ c[side] }}</span>
                </dd>
              </div>
            </dl>
          </CardContent>
          <CardFooter v-if="item.action === 'update'">
            <Button variant="outline" size="sm" as-child>
              <RouterLink :to="`${parentPath}/edit?undo=${item.number}`" :aria-label="`Undo version ${item.number}`">
                <Undo2Icon aria-hidden="true" />Undo this change
              </RouterLink>
            </Button>
          </CardFooter>
        </Card>
      </template>
      <template #empty>
        <EmptyState :icon="HistoryIcon" title="No changes yet" :description="`The history of this ${info.noun} starts with its next change.`" />
      </template>
    </PagedList>
  </main>
</template>
