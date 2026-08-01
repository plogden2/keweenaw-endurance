<template>
  <div class="live-event-qr" data-testid="rotator-live-qr">
    <canvas ref="canvasRef" class="live-event-qr__canvas" width="128" height="128" />
    <p class="live-event-qr__caption">view results at keweenawendurance.com</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import QRCode from 'qrcode'

const props = defineProps<{ url: string }>()
const canvasRef = ref<HTMLCanvasElement | null>(null)

async function renderQr() {
  const canvas = canvasRef.value
  if (!canvas || !props.url) return
  try {
    await QRCode.toCanvas(canvas, props.url, {
      width: 128,
      margin: 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#1a1a1a', light: '#ffffff' },
    })
  } catch {
    // Keep caption; board must not break if QR fails.
  }
}

onMounted(() => {
  void renderQr()
})
watch(
  () => props.url,
  () => {
    void renderQr()
  },
)
</script>

<style scoped>
.live-event-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  padding: 0.5rem;
  background: #fff;
  border-radius: 6px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
  max-width: 10rem;
}
.live-event-qr__canvas {
  display: block;
  width: 128px;
  height: 128px;
}
.live-event-qr__caption {
  margin: 0;
  font-size: 0.7rem;
  line-height: 1.25;
  text-align: center;
  color: #1a1a1a;
}
</style>
