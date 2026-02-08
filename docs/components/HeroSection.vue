<template>
  <section class="relative overflow-hidden px-6 sm:px-10 lg:px-16 xl:px-24 hero-grid min-h-[70vh] lg:min-h-screen flex items-center py-24 lg:py-0">
    <div class="edge-glow-top"/>
    <div class="mx-auto w-full max-w-7xl flex flex-col lg:flex-row items-center gap-10 lg:gap-10">
      <!-- Left: Text content -->
      <div class="flex-1 min-w-0 text-center lg:text-left">
        <h1 class="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-black dark:text-white mb-6">
          Aegis
        </h1>
        <p class="text-lg sm:text-xl text-gray-600 dark:text-gray-400 max-w-lg mx-auto lg:mx-0 mb-8">
          The most robust authentication framework for Go developers. Simple, modular, and built for performance.
        </p>
        <div class="flex flex-wrap gap-3 justify-center lg:justify-start">
          <UButton
            to="/get-started/installation"
            class="btn-primary translate-z-0"
          >
            Get started
            <span class="i-lucide-arrow-right" />
          </UButton>
        </div>
      </div>

      <!-- Right: Terminal -->
      <div class="flex-1 min-w-0 w-full max-w-xl lg:max-w-none">
        <HeroTerminal />
      </div>
    </div>
  </section>
</template>
