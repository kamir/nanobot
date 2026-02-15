<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useModeStore } from '../stores/mode'

const router = useRouter()
const modeStore = useModeStore()
const activating = ref(false)
const error = ref('')

const modes = [
  {
    id: 'standalone' as const,
    title: 'Standalone Desktop',
    description: 'Personal AI assistant. No Kafka, no group collaboration. Local-only agent loop with full dashboard.',
    icon: '&#x1f4bb;',
    features: ['Local agent loop', 'Timeline & dashboard', 'WhatsApp optional', 'Offline-capable'],
  },
  {
    id: 'full' as const,
    title: 'Group Master Desktop',
    description: 'Full local gateway with group collaboration and multi-agent orchestration via Kafka.',
    icon: '&#x1f310;',
    features: ['All channels active', 'Kafka group link', 'Agent orchestrator', 'Zone management'],
  },
  {
    id: 'remote' as const,
    title: 'Remote Client',
    description: 'Connect to a headless GoMikroBot agent running on a server. Thin client mode.',
    icon: '&#x2601;',
    features: ['Remote connection', 'Bearer token auth', 'Server dashboard', 'No local binary'],
  },
]

async function selectMode(mode: 'full' | 'standalone' | 'remote') {
  if (activating.value) return
  activating.value = true
  error.value = ''

  if (mode === 'remote') {
    // Remote: save mode, stay in Vue renderer for connection setup
    await modeStore.setMode(mode)
    activating.value = false
    router.push('/remote')
    return
  }

  // Local modes: call activate which starts sidecar + navigates to Go timeline.
  // The main process takes over the window — this Vue app will be unloaded.
  if (window.electronAPI) {
    const result = await window.electronAPI.mode.activate(mode)
    if (!result.ok) {
      error.value = result.error || 'Failed to start gateway'
      activating.value = false
    }
    // On success the window navigates away — no further code runs here.
  }
}
</script>

<template>
  <div class="picker-container">
    <div class="picker-header">
      <h1>GoMikroBot</h1>
      <p>Choose your operation mode</p>
    </div>
    <div class="picker-cards">
      <div
        v-for="mode in modes"
        :key="mode.id"
        class="mode-card"
        :class="{ disabled: activating }"
        @click="selectMode(mode.id)"
      >
        <div class="card-icon" v-html="mode.icon"></div>
        <h2>{{ mode.title }}</h2>
        <p class="card-desc">{{ mode.description }}</p>
        <ul class="card-features">
          <li v-for="f in mode.features" :key="f">{{ f }}</li>
        </ul>
      </div>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>

<style scoped>
.picker-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 40px;
}

.picker-header {
  text-align: center;
  margin-bottom: 48px;
}

.picker-header h1 {
  font-size: 28px;
  color: #58a6ff;
  margin-bottom: 8px;
}

.picker-header p {
  color: #8b949e;
  font-size: 14px;
}

.picker-cards {
  display: flex;
  gap: 24px;
  max-width: 1000px;
}

.mode-card {
  flex: 1;
  padding: 32px 24px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.mode-card:hover:not(.disabled) {
  border-color: #58a6ff;
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.mode-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.card-icon {
  font-size: 32px;
  margin-bottom: 16px;
}

.mode-card h2 {
  font-size: 16px;
  color: #f0f6fc;
  margin-bottom: 8px;
}

.card-desc {
  font-size: 12px;
  color: #8b949e;
  line-height: 1.5;
  margin-bottom: 16px;
}

.card-features {
  list-style: none;
  padding: 0;
}

.card-features li {
  font-size: 11px;
  color: #7ee787;
  padding: 3px 0;
}

.card-features li::before {
  content: '+ ';
  color: #3fb950;
}

.error {
  margin-top: 24px;
  color: #f85149;
  font-size: 13px;
}
</style>
