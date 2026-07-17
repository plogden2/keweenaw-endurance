export const DEFAULT_CATEGORY_COLORS: Record<string, string> = {
  advanced_men: '#1a3f3d',
  advanced_women: '#2f6b5a',
  beginner_men: '#9b654e',
  beginner_women: '#a1b383',
}

const FALLBACK_PALETTE = ['#1a3f3d', '#2f6b5a', '#9b654e', '#a1b383', '#6b7a76']

export function resolveCategoryColor(key: string, _apiColor?: string): string {
  if (DEFAULT_CATEGORY_COLORS[key]) return DEFAULT_CATEGORY_COLORS[key]
  let hash = 0
  for (let i = 0; i < key.length; i++) hash = (hash + key.charCodeAt(i) * (i + 1)) % 2147483647
  return FALLBACK_PALETTE[Math.abs(hash) % FALLBACK_PALETTE.length]
}
