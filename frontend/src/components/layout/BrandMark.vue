<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number
    withGlow?: boolean
  }>(),
  { size: 34, withGlow: false },
)
</script>

<template>
  <span class="brand-mark" :class="{ glow: withGlow }" :style="{ width: size + 'px', height: size + 'px' }">
    <svg viewBox="0 0 48 48" fill="none" aria-hidden="true">
      <defs>
        <linearGradient id="bm-teal" x1="0" y1="12" x2="0" y2="36" gradientUnits="userSpaceOnUse">
          <stop offset="0" stop-color="#a5ffed" />
          <stop offset="0.5" stop-color="#5be7c8" />
          <stop offset="1" stop-color="#25bfa0" />
        </linearGradient>
        <linearGradient id="bm-indigo" x1="0" y1="12" x2="0" y2="36" gradientUnits="userSpaceOnUse">
          <stop offset="0" stop-color="#b8c4ff" />
          <stop offset="0.5" stop-color="#8fa3ff" />
          <stop offset="1" stop-color="#6179ef" />
        </linearGradient>
        <radialGradient id="bm-glow" cx="50%" cy="50%" r="50%">
          <stop offset="0" stop-color="#5be7c8" stop-opacity="0.2" />
          <stop offset="1" stop-color="#5be7c8" stop-opacity="0" />
        </radialGradient>
      </defs>

      <rect x="1.25" y="1.25" width="45.5" height="45.5" rx="13" fill="var(--surface-2)" />
      <rect
        x="1.25"
        y="1.25"
        width="45.5"
        height="45.5"
        rx="13"
        stroke="var(--border-strong)"
        stroke-width="1.5"
      />
      <circle cx="24" cy="24" r="17" fill="url(#bm-glow)" />

      <g stroke-linecap="round">
        <path d="M14.6 34.6V13.8" stroke="url(#bm-teal)" stroke-width="5.2" />
        <path d="M33.4 34.6V13.8" stroke="url(#bm-indigo)" stroke-width="5.2" />
      </g>
      <path d="M17.4 17.3 30.6 30.4" stroke="#bffcf1" stroke-opacity="0.42" stroke-width="1.2" stroke-linecap="round" />

      <circle cx="19.7" cy="20.2" r="2.35" fill="#0b171d" stroke="#5be7c8" stroke-width="1.05" />
      <circle cx="19.7" cy="20.2" r="0.72" fill="#a5ffed" />
      <circle cx="24" cy="24.7" r="3.15" fill="#0b171d" stroke="#d8fff8" stroke-width="1.25" />
      <circle cx="24" cy="24.7" r="0.95" fill="#ffffff" />
      <circle cx="28.3" cy="29.2" r="2.35" fill="#0b171d" stroke="#8fa3ff" stroke-width="1.05" />
      <circle cx="28.3" cy="29.2" r="0.72" fill="#dfe5ff" />
    </svg>
  </span>
</template>

<style scoped>
.brand-mark {
  display: inline-flex;
  flex: none;
  border-radius: 13px;
}

.brand-mark svg {
  width: 100%;
  height: 100%;
  display: block;
}

.glow {
  filter: drop-shadow(0 0 16px rgba(86, 224, 192, 0.2));
}
</style>
