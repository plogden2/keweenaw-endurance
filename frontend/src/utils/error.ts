function axiosResponseData(err: object): unknown {
  if (!('response' in err)) return undefined
  const response = (err as { response?: { data?: unknown } }).response
  return response?.data
}

function axiosStatus(err: object): number | undefined {
  if (!('response' in err)) return undefined
  const response = (err as { response?: { status?: number } }).response
  const status = response?.status
  return typeof status === 'number' ? status : undefined
}

function formatWithStatus(message: string, status?: number): string {
  if (!status || message.includes(String(status))) return message
  return `${message} (${status})`
}

/**
 * Prefer API JSON `{ error }` from Axios failures so operators see bridge/hardware
 * detail instead of only "Request failed with status code NNN".
 */
export function getErrorMessage(err: unknown, fallback: string): string {
  if (err && typeof err === 'object') {
    const data = axiosResponseData(err)
    const status = axiosStatus(err)

    if (data && typeof data === 'object' && data !== null && 'error' in data) {
      const msg = (data as { error: unknown }).error
      if (typeof msg === 'string' && msg.trim()) {
        return formatWithStatus(msg.trim(), status)
      }
    }

    if (typeof data === 'string' && data.trim()) {
      return formatWithStatus(data.trim(), status)
    }

    if (err instanceof Error && err.message.trim()) {
      return err.message.trim()
    }
  }

  if (err instanceof Error && err.message.trim()) {
    return err.message.trim()
  }

  return fallback
}
