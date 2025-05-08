<template>
  <div class="ws-status-indicator" :class="statusClass" :title="statusText" @click="toggleDisplayText">
    <i class="el-icon-connection"></i>
    <span v-if="displayText">{{ statusText }}</span>
  </div>
</template>

<script>
export default {
  name: 'WsStatusIndicator',
  data() {
    return {
      displayText: false,
      checkInterval: null
    }
  },
  computed: {
    isConnected() {
      return this.$store.state.wsConnected
    },
    isConnecting() {
      return this.$store.state.wsConnecting
    },
    lastError() {
      return this.$store.state.wsLastError
    },
    statusClass() {
      if (this.isConnected) {
        return 'connected'
      } else if (this.isConnecting) {
        return 'connecting'
      } else {
        return 'disconnected'
      }
    },
    statusText() {
      if (this.isConnected) {
        return '连接正常'
      } else if (this.isConnecting) {
        return '正在连接...'
      } else {
        return this.lastError ? `连接断开: ${this.lastError}` : '连接断开'
      }
    }
  },
  methods: {
    toggleDisplayText() {
      this.displayText = !this.displayText
    },
    checkConnectionStatus() {
      this.$store.dispatch('checkWebSocketHealth')
    }
  },
  mounted() {
    // 每60秒检查一次连接状态
    this.checkInterval = setInterval(() => {
      this.checkConnectionStatus()
    }, 60000)
  },
  beforeDestroy() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval)
    }
  }
}
</script>

<style scoped>
.ws-status-indicator {
  position: fixed;
  top: 10px;
  right: 10px;
  z-index: 1000;
  padding: 5px 10px;
  border-radius: 15px;
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 5px;
  transition: all 0.3s ease;
  opacity: 0.7;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.ws-status-indicator:hover {
  opacity: 1;
}

.ws-status-indicator.connected {
  background-color: #67C23A;
  color: white;
}

.ws-status-indicator.connecting {
  background-color: #E6A23C;
  color: white;
  animation: pulse 1.5s infinite;
}

.ws-status-indicator.disconnected {
  background-color: #F56C6C;
  color: white;
}

@keyframes pulse {
  0% {
    opacity: 0.5;
  }
  50% {
    opacity: 1;
  }
  100% {
    opacity: 0.5;
  }
}
</style> 