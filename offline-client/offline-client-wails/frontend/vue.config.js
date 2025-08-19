module.exports = {
    publicPath: './', // 设置为相对路径，适用于Wails
    devServer: {
        port: 8090,
        open: true
        // 移除了proxy配置，因为在Wails环境下不需要
    }
}
