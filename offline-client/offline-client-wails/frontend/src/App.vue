<template>
    <div id="app">
        <ws-status-indicator v-if="isLoggedIn"></ws-status-indicator>
        <router-view />
        <button v-if="showHomeButton" @click="goHome" class="home-button">
            主页
        </button>
    </div>
</template>

<script>
import WsStatusIndicator from './components/WsStatusIndicator.vue'

export default {
    name: 'App',
    components: {
        WsStatusIndicator
    },
    computed: {
        isLoggedIn() {
            return this.$store.getters.isLoggedIn
        },
        showHomeButton() {
            return this.isLoggedIn && this.$route.path !== '/dashboard'
        }
    },
    methods: {
        goHome() {
            this.$router.push('/dashboard')
        }
    }
}
</script>

<style>
#app {
    font-family: 'Microsoft YaHei', Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    color: #2c3e50;
    height: 100vh;
    width: 100vw;
}

body,
html {
    margin: 0;
    padding: 0;
    height: 100%;
    width: 100%;
}

.container {
    width: 100%;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

.home-button {
    position: fixed;
    bottom: 40px;
    right: 40px;
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background-color: #409eff;
    color: white;
    border: none;
    font-size: 16px;
    cursor: pointer;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
}

.home-button:hover {
    background-color: #66b1ff;
}
</style>