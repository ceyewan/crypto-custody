<template>
    <div id="app">
        <router-view />
    </div>
</template>

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
</style>