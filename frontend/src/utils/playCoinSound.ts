import coinSoundUrl from '@/assets/audio/Mario-coin-sound.mp3'
import { rfidApi } from '@/services/api'

function playBrowserCoinSound(): void {
  try {
    const audio = new Audio(coinSoundUrl)
    void audio.play().catch(() => {
      /* autoplay may be blocked; no UI label */
    })
  } catch {
    /* ignore missing audio */
  }
}

/**
 * Play the RFID tap coin tone.
 * When `bridgeOnline` is true, uses the local bridge beep (same as chip reads).
 * When false, uses browser audio. When omitted, tries the bridge then falls back.
 */
export function playCoinSound(options?: { bridgeOnline?: boolean }): void {
  if (options?.bridgeOnline === true) {
    void rfidApi.playLocalBridgeBeep()
    return
  }
  if (options?.bridgeOnline === false) {
    playBrowserCoinSound()
    return
  }
  void rfidApi.playLocalBridgeBeep().then((ok) => {
    if (!ok) playBrowserCoinSound()
  })
}
