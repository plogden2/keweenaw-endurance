<template>
  <div class="live-event-qr" data-testid="rotator-live-qr">
    <canvas ref="canvasRef" class="live-event-qr__canvas" width="96" height="96" />
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
      width: 96,
      // Card padding supplies the quiet zone; keep library margin minimal.
      margin: 1,
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
  gap: 0.15rem;
  padding: 0.2rem 0.25rem 0.25rem;
  background: #fff;
  border-radius: 6px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
  max-width: 7.5rem;
}
.live-event-qr__canvas {
  display: block;
  width: 96px;
  height: 96px;
}
.live-event-qr__caption {
  margin: 0;
  font-size: 0.62rem;
  line-height: 1.15;
  text-align: center;
  color: #1a1a1a;
}
</style>
