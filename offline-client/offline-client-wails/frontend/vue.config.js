module.exports = {
    publicPath: './', // 设置为相对路径，适用于Wails
    devServer: {
        port: 8090,
        open: true,
        proxy: {
            '/api': {
                target: 'https://crypto-custody-offline-server.ceyewan.icu',
                ws: true,
                changeOrigin: true,
                pathRewrite: {
                    '^/api': ''
                }
            },
            '/ws': {
                target: 'wss://crypto-custody-offline-server.ceyewan.icu',
                ws: true,
                changeOrigin: true,
                pathRewrite: {
                    '^/ws': '/ws'
                }
            }
        }
    }
}
