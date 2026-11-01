<script setup lang="ts">
import BrandMark from './BrandMark.vue'
</script>

<template>
  <div class="boot-screen">
    <div class="boot-glow" />
    <div class="boot-inner">
      <BrandMark :size="58" with-glow />
      <h1>NPC 客户端</h1>
      <p>正在载入本机连接状态</p>
      <div class="boot-track"><span /></div>
    </div>
    <div class="boot-meta mono">nps enhanced · npc supervisor</div>
  </div>
</template>

<style scoped>
.boot-screen {
  position: relative;
  min-height: 100vh;
  display: grid;
  place-items: center;
  overflow: hidden;
}

.boot-screen::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(120, 160, 190, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(120, 160, 190, 0.045) 1px, transparent 1px);
  background-size: 42px 42px;
  mask-image: radial-gradient(ellipse at center, black, transparent 72%);
}

.boot-glow {
  position: absolute;
  width: 560px;
  height: 560px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(86, 224, 192, 0.12), transparent 66%);
  filter: blur(10px);
}

.boot-inner {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  animation: fade-up 420ms var(--ease);
}

.boot-inner h1 {
  margin-top: 8px;
  font-size: 24px;
  letter-spacing: -0.03em;
}

.boot-inner p {
  color: var(--text-3);
  font-size: 12.5px;
}

.boot-track {
  width: 180px;
  height: 2px;
  margin-top: 17px;
  border-radius: 2px;
  background: var(--surface-3);
  overflow: hidden;
}

.boot-track span {
  display: block;
  width: 44%;
  height: 100%;
  border-radius: 2px;
  background: linear-gradient(90deg, transparent, var(--accent), transparent);
  animation: boot-slide 1.2s var(--ease) infinite;
}

.boot-meta {
  position: absolute;
  bottom: 26px;
  color: var(--text-4);
  font-size: 10.5px;
  letter-spacing: 0.08em;
}

@keyframes boot-slide {
  from {
    transform: translateX(-110%);
  }
  to {
    transform: translateX(340%);
  }
}

@keyframes fade-up {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
