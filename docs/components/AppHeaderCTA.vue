<script setup lang="ts">
const route = useRoute()

const links = [
  { label: "Landing", to: "/" },
  { label: 'Docs', to: '/get-started/introduction' },
  { label: 'Changelog', to: '/changelog' },
  { label: "Communities", to: "/communities"}

]

const isActive = (to: string) => {
  if (to === '/') return route.path === '/'
  return route.path.startsWith(to)
}
</script>

<template>
  <div class="flex items-center gap-2 sm:gap-4">
    <!-- Search Trigger (Minimal) -->
    <AppSearch class="hidden md:hidden lg:hidden sm:flex" />

    <NuxtLink
      v-for="link in links"
      :key="link.to"
      :to="link.to"
      class="sm:hidden md:flex lg:flex px-4 py-2 text-sm font-medium transition-all duration-200 border-b-2"
      :class="[
        isActive(link.to) 
          ? 'text-blue-500 border-blue-500 bg-blue-500/5' 
          : 'text-gray-500 border-transparent hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-white/5'
      ]"
    >
      {{ link.label }}
    </NuxtLink>
  </div>
</template>
