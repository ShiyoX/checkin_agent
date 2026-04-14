<template>
  <div>
    <!-- 悬浮按钮 -->
    <button
      @click="toggleChat"
      class="fixed bottom-6 right-6 p-4 rounded-full bg-blue-600 text-white shadow-lg hover:bg-blue-700 transition-colors z-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
    >
      <i class="fas" :class="isOpen ? 'fa-times' : 'fa-robot'"></i>
    </button>

    <!-- 聊天窗口 -->
    <transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="transform translate-y-4 opacity-0"
      enter-to-class="transform translate-y-0 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="transform translate-y-0 opacity-100"
      leave-to-class="transform translate-y-4 opacity-0"
    >
      <div
        v-if="isOpen"
        class="fixed bottom-24 right-6 w-80 sm:w-96 h-[32rem] bg-white rounded-2xl shadow-2xl flex flex-col z-50 overflow-hidden border border-gray-100"
      >
        <!-- 头部 -->
        <div class="bg-blue-600 p-4 text-white flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <i class="fas fa-robot text-xl"></i>
            <span class="font-semibold">智能签到助手</span>
          </div>
        </div>

        <!-- 聊天区域 -->
        <div
          class="flex-1 p-4 overflow-y-auto bg-gray-50 space-y-4"
          ref="chatContainer"
        >
          <div
            v-for="(msg, index) in messages"
            :key="index"
            class="flex"
            :class="msg.role === 'user' ? 'justify-end' : 'justify-start'"
          >
            <!-- 机器人头像 -->
            <div
              v-if="msg.role === 'assistant'"
              class="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center mr-2 flex-shrink-0"
            >
              <i class="fas fa-robot text-blue-600 text-sm"></i>
            </div>

            <!-- 消息气泡 -->
            <div
              class="max-w-[75%] rounded-2xl px-4 py-2 text-sm shadow-sm"
              :class="[
                msg.role === 'user'
                  ? 'bg-blue-600 text-white rounded-tr-none'
                  : 'bg-white text-gray-800 rounded-tl-none border border-gray-100 whitespace-pre-wrap',
                msg.isLoading ? 'animate-pulse' : ''
              ]"
            >
              <template v-if="msg.isLoading">
                <div class="flex space-x-1">
                  <div class="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                  <div class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0.1s"></div>
                  <div class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0.2s"></div>
                </div>
              </template>
              <template v-else>
                {{ msg.content }}
              </template>
            </div>
          </div>
        </div>

        <!-- 输入区域 -->
        <div class="p-3 border-t border-gray-100 bg-white">
          <form @submit.prevent="sendMessage" class="flex space-x-2">
            <input
              v-model="inputText"
              type="text"
              placeholder="问问你的签到情况..."
              class="flex-1 px-4 py-2 rounded-full border border-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm bg-gray-50"
              :disabled="isLoading"
            />
            <button
              type="submit"
              class="p-2 rounded-full bg-blue-600 text-white w-10 h-10 flex items-center justify-center hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-shrink-0"
              :disabled="isLoading || !inputText.trim()"
            >
              <i class="fas fa-paper-plane text-sm"></i>
            </button>
          </form>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { toast } from 'vue3-toastify'
import { http } from '@/api'

const isOpen = ref(false)
const inputText = ref('')
const isLoading = ref(false)
const chatContainer = ref<HTMLElement | null>(null)

interface Message {
  role: 'user' | 'assistant'
  content: string
  isLoading?: boolean
}

const messages = ref<Message[]>([
  {
    role: 'assistant',
    content: '你好！我是签到助手。你可以问我：\n- 我今天签到了吗？\n- 帮我查一下我这个月的签到记录\n- 我的积分还有多少？'
  }
])

const toggleChat = () => {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    scrollToBottom()
  }
}

const scrollToBottom = async () => {
  await nextTick()
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}

const sendMessage = async () => {
  const text = inputText.value.trim()
  if (!text || isLoading.value) return

  // 添加用户消息
  messages.value.push({
    role: 'user',
    content: text
  })
  
  inputText.value = ''
  isLoading.value = true
  scrollToBottom()

  // 添加 loading 占位
  messages.value.push({
    role: 'assistant',
    content: '',
    isLoading: true
  })
  scrollToBottom()

  try {
    const response = await http.post('/agent/chat', { message: text }, { timeout: 120000 } as any)

    messages.value.pop()

    messages.value.push({
      role: 'assistant',
      content: (response.data as any).reply || '抱歉，我没有理解你的意思。'
    })
  } catch (error: any) {
    if (messages.value[messages.value.length - 1]?.isLoading) {
      messages.value.pop()
    }

    messages.value.push({
      role: 'assistant',
      content: '对不起，AI助手暂时无法响应，请稍后再试。'
    })
  } finally {
    isLoading.value = false
    scrollToBottom()
  }
}
</script>
