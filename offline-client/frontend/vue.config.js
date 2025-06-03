module.exports = {
    publicPath: './', // 设置为相对路径，适用于Electron
    devServer: {
        port: 8090,
        open: true,
        proxy: {
            '/api': {
                target: 'http://localhost:8080',
                changeOrigin: true,
                pathRewrite: {
                    '^/api': ''
                }
            }
        }
    }
}
