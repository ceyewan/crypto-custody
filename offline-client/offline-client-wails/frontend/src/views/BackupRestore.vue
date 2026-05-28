<template>
    <div class="page backup-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">备份恢复</h2>
                <p class="page-subtitle">第一阶段提供数据库热备份下载，恢复操作保留为管理员线下确认步骤。</p>
            </div>
        </div>

        <el-card>
            <el-descriptions :column="1" border>
                <el-descriptions-item label="热备份">
                    下载当前离线服务端 SQLite 数据库快照，用于本机或内网备份。
                </el-descriptions-item>
                <el-descriptions-item label="冷备份">
                    可将下载文件写入加密介质后离线保管；恢复前需要人工校验来源和哈希。
                </el-descriptions-item>
                <el-descriptions-item label="恢复">
                    当前不在客户端直接执行覆盖恢复，避免误操作破坏正在运行的 MPC 会话。
                </el-descriptions-item>
            </el-descriptions>

            <div class="actions">
                <el-button type="primary" icon="el-icon-download" :loading="downloading" @click="downloadBackup">
                    下载热备份
                </el-button>
            </div>
        </el-card>
    </div>
</template>

<script>
export default {
    name: 'BackupRestore',
    data() {
        return {
            downloading: false
        }
    },
    methods: {
        async downloadBackup() {
            this.downloading = true
            try {
                const response = await this.$offlineApi.downloadBackup()
                const header = response.headers['content-disposition'] || ''
                const matched = header.match(/filename="?([^"]+)"?/)
                const fileName = matched ? matched[1] : `offline-backup-${this.timestamp()}.db`
                const blob = new Blob([response.data], { type: 'application/octet-stream' })
                const url = URL.createObjectURL(blob)
                const link = document.createElement('a')
                link.href = url
                link.download = fileName
                link.click()
                URL.revokeObjectURL(url)
                this.$message.success('备份文件已下载')
            } catch (error) {
                this.$message.error(error.response?.data?.error || error.message || '下载备份失败')
            } finally {
                this.downloading = false
            }
        },
        timestamp() {
            const d = new Date()
            const pad = value => String(value).padStart(2, '0')
            return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
        }
    }
}
</script>

<style scoped>
.actions {
    margin-top: 18px;
}
</style>
