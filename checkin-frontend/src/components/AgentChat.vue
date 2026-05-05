<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const router = useRouter()
const route = useRoute()

const openAssistant = () => {
  if (route.name !== 'assistant') {
    router.push({ name: 'assistant' })
  }
}
</script>

<template>
  <button
    type="button"
    class="fixed bottom-6 right-6 z-50 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-600 via-cyan-500 to-teal-400 text-white shadow-[0_18px_50px_rgba(14,116,144,0.35)] transition duration-200 hover:-translate-y-1 hover:shadow-[0_24px_60px_rgba(14,116,144,0.42)] focus:outline-none focus:ring-4 focus:ring-cyan-200"
    aria-label="打开签到助手"
    @click="openAssistant"
  >
    <span class="absolute inset-0 rounded-2xl border border-white/25"></span>
    <i class="fas fa-robot text-2xl"></i>
  </button>
</template>
