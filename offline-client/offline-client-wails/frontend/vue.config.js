const devServerHttpUrl = process.env.VUE_APP_DEV_SERVER_HTTP_URL || 'http://127.0.0.1:8080'
const devServerWsUrl = process.env.VUE_APP_DEV_SERVER_WS_URL ||
    devServerHttpUrl.replace(/^https:\/\//i, 'wss://').replace(/^http:\/\//i, 'ws://')

module.exports = {
    publicPath: './', // 设置为相对路径，适用于Wails
    devServer: {
        port: 8090,
        open: true,
        proxy: {
            '/api': {
                target: devServerHttpUrl,
                ws: true,
                changeOrigin: true,
                pathRewrite: {
                    '^/api': ''
                }
            },
            '/ws': {
                target: devServerWsUrl,
                ws: true,
                changeOrigin: true,
                pathRewrite: {
                    '^/ws': '/ws'
                }
            }
        }
    }
}
