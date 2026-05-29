import html2canvas from 'html2canvas'

export function slugifyFilenamePart(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

export function buildImageFilename(label: string, suffix: string): string {
  const slug = slugifyFilenamePart(label)
  return `${slug || 'image'}-${suffix}.png`
}

export function buildCertificateFilename(
  eventName: string,
  participantName: string,
  suffix: string,
): string {
  const eventSlug = slugifyFilenamePart(eventName) || 'event'
  const participantSlug = slugifyFilenamePart(participantName) || 'participant'
  return `${eventSlug}-${participantSlug}-${suffix}.png`
}

export interface SaveImageOptions {
  backgroundColor?: string
  scale?: number
  width?: number
  height?: number
}

export async function saveElementAsImage(
  element: HTMLElement,
  filename: string,
  options: SaveImageOptions = {},
): Promise<void> {
  const {
    backgroundColor = '#ffffff',
    scale = 2,
    width,
    height,
  } = options

  const canvas = await html2canvas(element, {
    backgroundColor,
    scale,
    useCORS: true,
    width,
    height,
  })

  const link = document.createElement('a')
  link.download = filename
  link.href = canvas.toDataURL('image/png')
  link.click()
}
