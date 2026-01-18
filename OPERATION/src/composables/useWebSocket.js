import { ref, onUnmounted } from 'vue'

export function useWebSocket(url) {
  const ws = ref(null)
  const isConnected = ref(false)
  const lastMessage = ref(null)
  const error = ref(null)

  function connect() {
    const token = localStorage.getItem('token')
    const wsUrl = `${url}?token=${token}`

    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      isConnected.value = true
      error.value = null
      console.log('WebSocket connected')
    }

    ws.value.onmessage = (event) => {
      try {
        lastMessage.value = JSON.parse(event.data)
      } catch {
        lastMessage.value = event.data
      }
    }

    ws.value.onerror = (e) => {
      error.value = e
      console.error('WebSocket error:', e)
    }

    ws.value.onclose = () => {
      isConnected.value = false
      // Attempt reconnection after 5 seconds
      setTimeout(() => {
        if (!isConnected.value) {
          connect()
        }
      }, 5000)
    }
  }

  function send(data) {
    if (ws.value && isConnected.value) {
      ws.value.send(typeof data === 'string' ? data : JSON.stringify(data))
    }
  }

  function disconnect() {
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    isConnected,
    lastMessage,
    error,
    connect,
    send,
    disconnect
  }
}
