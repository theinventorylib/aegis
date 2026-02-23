<template>
  <ClientOnly v-if="language === 'mermaid'">
    <div class="mermaid-container flex justify-center py-4 overflow-x-auto" v-html="svgCode"></div>
  </ClientOnly>
  <pre v-else :class="$props.class"><slot /></pre>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import mermaid from 'mermaid'
import { useColorMode } from '#imports'

const props = defineProps({
  code: {
    type: String,
    default: ""
  },
  language: {
    type: String,
    default: null
  },
  filename: {
    type: String,
    default: null
  },
  highlights: {
    type: Array,
    default: () => []
  },
  meta: {
    type: String,
    default: null
  },
  class: {
    type: String,
    default: null
  }
})

const svgCode = ref('')
const colorMode = useColorMode()

const renderMermaid = async () => {
  if (props.language === 'mermaid' && props.code) {
    try {
      mermaid.initialize({ 
        startOnLoad: false, 
        theme: colorMode.value === 'dark' ? 'dark' : 'default',
        securityLevel: 'loose'
      })
      // Use a unique ID for each diagram to prevent conflicts
      const id = 'mermaid-' + Math.random().toString(36).substring(2, 9)
      const { svg } = await mermaid.render(id, props.code)
      svgCode.value = svg
    } catch (e) {
      console.error('Mermaid render error:', e)
      svgCode.value = `<div class="text-red-500 text-sm mb-2">Mermaid syntax error:</div><pre class="bg-gray-100 dark:bg-gray-800 p-4 rounded overflow-x-auto text-sm">${props.code}</pre>`
    }
  }
}

onMounted(() => {
  renderMermaid()
})

watch(() => colorMode.value, () => {
  renderMermaid()
})

watch(() => props.code, () => {
  renderMermaid()
})
</script>

<style>
pre code .line {
  display: block;
}
.mermaid-container svg {
  max-width: 100%;
  height: auto;
}
</style>
