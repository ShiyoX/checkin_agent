<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue3-toastify'
import { http } from '@/api'
import { useUserStore } from '@/stores/user'

interface Message {
  id: number
  role: 'user' | 'assistant'
  content: string
  createdAt: string
  isLoading?: boolean
}

interface ChatResponse {
  reply?: string
}

const router = useRouter()
const userStore = useUserStore()

const chatContainer = ref<HTMLElement | null>(null)
const inputText = ref('')
const isLoading = ref(false)
const messageId = ref(1)
const composerRows = ref(1)

const messages = ref<Message[]>([
  {
    id: messageId.value++,
    role: 'assistant',
    content:
      '你好，我是签到助手。你可以直接问我签到、补签、积分和本月记录，我会按当前账号的数据来回答。',
    createdAt: new Date().toISOString(),
  },
])

const quickPrompts = [
  '我今天签到了吗？',
  '帮我总结一下这个月的签到情况',
  '我现在还有多少积分？',
  '补签一次会消耗多少积分？',
]

const pageTitle = computed(() => {
  return userStore.currentUser?.username
    ? `${userStore.currentUser.username} 的签到助手`
    : '签到助手'
})

const canSend = computed(() => inputText.value.trim().length > 0 && !isLoading.value)

const formatTime = (value: string) =>
  new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })

const scrollToBottom = async () => {
  await nextTick()
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}

const syncComposerRows = () => {
  const lineBreaks = inputText.value.split('\n').length
  composerRows.value = Math.max(1, Math.min(6, lineBreaks))
}

const pushMessage = (message: Omit<Message, 'id' | 'createdAt'>) => {
  messages.value.push({
    id: messageId.value++,
    createdAt: new Date().toISOString(),
    ...message,
  })
}

const submitPrompt = async (text: string) => {
  const content = text.trim()
  if (!content || isLoading.value) {
    return
  }

  pushMessage({ role: 'user', content })
  inputText.value = ''
  composerRows.value = 1
  isLoading.value = true
  await scrollToBottom()

  pushMessage({ role: 'assistant', content: '', isLoading: true })
  await scrollToBottom()

  try {
    const response = await http.post<ChatResponse>('/agent/chat', { message: content })
    const reply = response.data?.reply?.trim() || '我没有拿到有效回复，你可以换个问法再试一次。'

    messages.value = messages.value.filter((item) => !item.isLoading)
    pushMessage({ role: 'assistant', content: reply })
  } catch (error) {
    messages.value = messages.value.filter((item) => !item.isLoading)
    pushMessage({
      role: 'assistant',
      content: '当前无法连接签到助手，请稍后再试。',
    })
    toast.error('签到助手暂时不可用')
  } finally {
    isLoading.value = false
    await scrollToBottom()
  }
}

const sendMessage = async () => {
  await submitPrompt(inputText.value)
}

const handleQuickPrompt = async (prompt: string) => {
  await submitPrompt(prompt)
}

const handleKeydown = async (event: KeyboardEvent) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    await sendMessage()
  }
}

watch(inputText, () => {
  syncComposerRows()
})

onMounted(() => {
  scrollToBottom()
})
</script>

<template>
  <div
    class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.18),_transparent_34%),linear-gradient(180deg,_#f4fbff_0%,_#eef6ff_48%,_#f8fafc_100%)]"
  >
    <div class="mx-auto flex min-h-screen max-w-7xl gap-6 px-4 py-4 sm:px-6 lg:px-8">
      <aside
        class="hidden w-80 shrink-0 rounded-[32px] border border-white/70 bg-white/75 p-6 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur xl:flex xl:flex-col"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-600 to-cyan-400 text-white shadow-lg"
          >
            <i class="fas fa-robot text-xl"></i>
          </div>
          <div>
            <p class="text-sm font-medium uppercase tracking-[0.24em] text-cyan-700">Checkin AI</p>
            <h1 class="text-2xl font-semibold text-slate-900">{{ pageTitle }}</h1>
          </div>
        </div>

        <div class="mt-8 rounded-[28px] bg-slate-950 px-5 py-6 text-slate-100">
          <p class="text-xs uppercase tracking-[0.24em] text-cyan-300">可处理的问题</p>
          <ul class="mt-4 space-y-3 text-sm leading-6 text-slate-300">
            <li>签到状态和月历记录查询</li>
            <li>补签规则和积分消耗说明</li>
            <li>积分余额与奖励获取建议</li>
            <li>结合当前账号返回个性化答复</li>
          </ul>
        </div>

        <div class="mt-6 rounded-[28px] border border-slate-200 bg-slate-50 p-5">
          <p class="text-sm font-semibold text-slate-900">快捷提问</p>
          <div class="mt-4 flex flex-wrap gap-3">
            <button
              v-for="prompt in quickPrompts"
              :key="prompt"
              type="button"
              class="rounded-full border border-slate-200 bg-white px-4 py-2 text-left text-sm text-slate-700 transition hover:border-cyan-300 hover:text-cyan-700"
              :disabled="isLoading"
              @click="handleQuickPrompt(prompt)"
            >
              {{ prompt }}
            </button>
          </div>
        </div>

        <button
          type="button"
          class="mt-auto inline-flex items-center gap-2 text-sm font-medium text-slate-500 transition hover:text-slate-900"
          @click="router.push({ name: 'home' })"
        >
          <i class="fas fa-arrow-left"></i>
          返回签到首页
        </button>
      </aside>

      <main
        class="flex min-h-[calc(100vh-2rem)] flex-1 flex-col overflow-hidden rounded-[32px] border border-white/70 bg-white/85 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur"
      >
        <header class="border-b border-slate-200/80 px-4 py-4 sm:px-6">
          <div class="flex items-center justify-between gap-4">
            <div class="flex min-w-0 items-center gap-3">
              <button
                type="button"
                class="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-slate-200 text-slate-600 transition hover:border-slate-300 hover:text-slate-900 xl:hidden"
                aria-label="返回首页"
                @click="router.push({ name: 'home' })"
              >
                <i class="fas fa-arrow-left"></i>
              </button>
              <div
                class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-600 to-cyan-400 text-white shadow-lg xl:hidden"
              >
                <i class="fas fa-robot"></i>
              </div>
              <div class="min-w-0">
                <h2 class="truncate text-lg font-semibold text-slate-900">{{ pageTitle }}</h2>
                <p class="text-sm text-slate-500">像 ChatGPT 一样连续对话，但围绕你的签到数据工作。</p>
              </div>
            </div>

            <div class="hidden items-center gap-2 rounded-full bg-emerald-50 px-3 py-2 text-sm text-emerald-700 sm:flex">
              <span class="h-2.5 w-2.5 rounded-full bg-emerald-500"></span>
              在线
            </div>
          </div>
        </header>

        <section ref="chatContainer" class="flex-1 overflow-y-auto px-4 py-6 sm:px-6 lg:px-10">
          <div class="mx-auto flex w-full max-w-4xl flex-col gap-6">
            <div
              v-for="message in messages"
              :key="message.id"
              class="flex gap-4"
              :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
            >
              <template v-if="message.role === 'assistant'">
                <div
                  class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-600 to-cyan-400 text-white shadow-lg"
                >
                  <i class="fas fa-robot"></i>
                </div>
              </template>

              <article
                class="max-w-[85%] rounded-[28px] px-5 py-4 text-[15px] leading-7 shadow-sm sm:max-w-[75%]"
                :class="
                  message.role === 'user'
                    ? 'bg-slate-950 text-white'
                    : 'border border-slate-200 bg-white text-slate-800'
                "
              >
                <div v-if="message.isLoading" class="flex items-center gap-2 py-1">
                  <span class="h-2.5 w-2.5 animate-bounce rounded-full bg-slate-400"></span>
                  <span
                    class="h-2.5 w-2.5 animate-bounce rounded-full bg-slate-400"
                    style="animation-delay: 0.15s"
                  ></span>
                  <span
                    class="h-2.5 w-2.5 animate-bounce rounded-full bg-slate-400"
                    style="animation-delay: 0.3s"
                  ></span>
                </div>
                <p v-else class="whitespace-pre-wrap break-words">{{ message.content }}</p>
                <p
                  class="mt-3 text-xs"
                  :class="message.role === 'user' ? 'text-slate-300' : 'text-slate-400'"
                >
                  {{ formatTime(message.createdAt) }}
                </p>
              </article>

              <template v-if="message.role === 'user'">
                <div
                  class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-cyan-100 text-cyan-700"
                >
                  <i class="fas fa-user"></i>
                </div>
              </template>
            </div>
          </div>
        </section>

        <footer class="border-t border-slate-200/80 bg-white/80 px-4 py-4 sm:px-6 lg:px-10">
          <div class="mx-auto w-full max-w-4xl">
            <div class="mb-3 flex flex-wrap gap-2 xl:hidden">
              <button
                v-for="prompt in quickPrompts"
                :key="prompt"
                type="button"
                class="rounded-full border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 transition hover:border-cyan-300 hover:text-cyan-700"
                :disabled="isLoading"
                @click="handleQuickPrompt(prompt)"
              >
                {{ prompt }}
              </button>
            </div>

            <form
              class="rounded-[28px] border border-slate-200 bg-white p-3 shadow-[0_12px_40px_rgba(15,23,42,0.06)]"
              @submit.prevent="sendMessage"
            >
              <div class="flex items-end gap-3">
                <textarea
                  v-model="inputText"
                  class="max-h-48 min-h-[56px] flex-1 resize-none border-0 bg-transparent px-3 py-3 text-[15px] leading-7 text-slate-900 outline-none placeholder:text-slate-400"
                  :rows="composerRows"
                  :disabled="isLoading"
                  placeholder="输入你的问题。按 Enter 发送，Shift + Enter 换行。"
                  @keydown="handleKeydown"
                ></textarea>
                <button
                  type="submit"
                  class="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-950 text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-300"
                  :disabled="!canSend"
                >
                  <i class="fas fa-arrow-up"></i>
                </button>
              </div>
            </form>
          </div>
        </footer>
      </main>
    </div>
  </div>
</template>
