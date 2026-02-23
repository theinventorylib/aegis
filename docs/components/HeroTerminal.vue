<template>
  <div class="relative group mt-8 sm:mt-0 w-full">
    <div
      class="relative bg-black/90 rounded-xl border border-white/5 overflow-hidden shadow-2xl backdrop-blur-sm"
    >
      <div class="flex items-center justify-between bg-[#111] px-3 py-2 border-b border-white/5">
        <div class="flex gap-1.5">
          <span class="w-2.5 h-2.5 rounded-full bg-[#ff5f56]" />
          <span class="w-2.5 h-2.5 rounded-full bg-[#ffbd2e]" />
          <span class="w-2.5 h-2.5 rounded-full bg-[#27c93f]" />
        </div>
        <span class="font-mono text-xs text-gray-500">Terminal</span>
      </div>
      <div class="p-6 sm:p-8 font-mono text-[13px] leading-relaxed text-gray-300 overflow-x-auto">
        <div v-for="(line, i) in lines" :key="i">
          <template v-if="line.type === 'comment'">
            <span class="text-gray-500">{{ line.text }}</span>
          </template>
          <template v-else-if="line.type === 'command'">
            <span class="text-green-400 select-none">❯ </span>
            <span class="text-green-400">{{ line.cmd }}</span>
            <span class="text-gray-300"> {{ line.args }}</span>
          </template>
          <template v-else>
            <br />
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const lines = [
  { type: 'comment', text: '# Install Aegis core' },
  { type: 'command', cmd: 'go ', args: 'get github.com/theinventorylib/aegis' },
  { type: 'blank' },
  { type: 'comment', text: '# Install the CLI tool' },
  { type: 'command', cmd: 'go ', args: 'install github.com/theinventorylib/aegis/cmd/aegis@latest' },
  { type: 'blank' },
  { type: 'comment', text: '# Export migrations for your database' },
  { type: 'command', cmd: 'aegis ', args: 'migrate export --dialect postgres --output ./migrations' },
]
</script>
