export async function reloadWithSwActivation(): Promise<void> {
  const reg = await navigator.serviceWorker?.getRegistration()
  if (reg?.waiting) {
    navigator.serviceWorker.addEventListener('controllerchange', () => location.reload(), { once: true })
    reg.waiting.postMessage({ type: 'SKIP_WAITING' })
  } else {
    location.reload()
  }
}
