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
      checkInterval: null,
      tokenWarning: false
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
    token() {
      return this.$store.state.token
    },
    statusClass() {
      if (this.tokenWarning) {
        return 'warning'
      }
      if (this.isConnected) {
        return 'connected'
      } else if (this.isConnecting) {
        return 'connecting'
      } else {
        return 'disconnected'
      }
    },
    statusText() {
      if (this.tokenWarning) {
        return 'Token格式可能有问题'
      }
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
      if (!this.isConnected) {
        this.checkAuthStatus()
      }
    },
    checkConnectionStatus() {
      this.$store.dispatch('checkWebSocketHealth')
    },
    checkAuthStatus() {
      // 检查Token是否存在
      const token = localStorage.getItem('token')
      console.log('当前token状态:', {
        token: token ? `${token.substring(0, 10)}...` : null,
        storeToken: this.token ? `${this.token.substring(0, 10)}...` : null,
        hasToken: !!token,
        wsConnected: this.isConnected
      })

      // 检查token是否以Bearer开头，如果是需要修复(去掉前缀)
      if (token && token.startsWith('Bearer ')) {
        const fixedToken = token.substring(7)
        localStorage.setItem('token', fixedToken)
        this.$store.commit('setToken', fixedToken)
        console.log('已修复token格式 (移除Bearer前缀):', fixedToken.substring(0, 10) + '...')
        this.tokenWarning = true
      } else {
        this.tokenWarning = false
      }
    }
  },
  mounted() {
    // 初始检查token
    this.checkAuthStatus()

    // 每60秒检查一次连接状态
    this.checkInterval = setInterval(() => {
      this.checkConnectionStatus()
      this.checkAuthStatus()
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

.ws-status-indicator.warning {
  background-color: #F56C6C;
  color: white;
  animation: pulse 1s infinite;
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