<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useModeStore } from '../stores/mode'

const router = useRouter()
const modeStore = useModeStore()

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
    title: 'Full Desktop',
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
  await modeStore.setMode(mode)
  if (mode === 'remote') {
    router.push('/remote')
  } else {
    router.push('/dashboard')
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

.mode-card:hover {
  border-color: #58a6ff;
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
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
</style>
