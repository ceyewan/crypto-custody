<template>
    <div class="page backup-page">
        <div class="toolbar">
            <div>
                <h2>备份恢复</h2>
                <p>管理离线服务端 SQLite 数据库热备份、加密冷备份和恢复记录。</p>
            </div>
            <div class="actions">
                <el-button type="primary" icon="el-icon-document-copy" :loading="hotCreating" @click="createHot">
                    热备份
                </el-button>
                <el-button type="success" icon="el-icon-lock" @click="openColdDialog">
                    加密冷备份
                </el-button>
                <el-button icon="el-icon-refresh" :loading="loading" @click="fetchBackups">刷新</el-button>
            </div>
        </div>

        <div class="summary-grid">
            <div class="summary-item">
                <span class="summary-label">全部备份</span>
                <strong>{{ backups.length }}</strong>
            </div>
            <div class="summary-item">
                <span class="summary-label">热备份</span>
                <strong>{{ hotCount }}</strong>
            </div>
            <div class="summary-item">
                <span class="summary-label">加密冷备份</span>
                <strong>{{ coldCount }}</strong>
            </div>
            <div class="summary-item">
                <span class="summary-label">已恢复记录</span>
                <strong>{{ restoredCount }}</strong>
            </div>
        </div>

        <el-card class="table-card">
            <div slot="header" class="table-header">
                <span>备份列表</span>
                <el-input
                    v-model="keyword"
                    class="search-input"
                    size="small"
                    clearable
                    prefix-icon="el-icon-search"
                    placeholder="搜索编号或文件名"
                />
            </div>

            <el-table :data="filteredBackups" v-loading="loading" empty-text="暂无备份记录">
                <el-table-column prop="BackupNo" label="备份编号" min-width="180" show-overflow-tooltip />
                <el-table-column label="类型" width="120">
                    <template slot-scope="scope">
                        <el-tag v-if="scope.row.Encrypted" type="success" size="small">冷备份</el-tag>
                        <el-tag v-else size="small">热备份</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="FileName" label="文件名" min-width="210" show-overflow-tooltip />
                <el-table-column label="状态" width="120">
                    <template slot-scope="scope">
                        <el-tag :type="statusType(scope.row.Status)" size="small">{{ statusText(scope.row.Status) }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="CreatedBy" label="创建人" width="120" />
                <el-table-column label="创建时间" width="170">
                    <template slot-scope="scope">{{ formatTime(scope.row.CreatedAt || scope.row.created_at) }}</template>
                </el-table-column>
                <el-table-column prop="FileHash" label="文件哈希" min-width="240" show-overflow-tooltip />
                <el-table-column label="操作" width="280" fixed="right">
                    <template slot-scope="scope">
                        <el-button size="mini" icon="el-icon-finished" @click="verify(scope.row)">校验</el-button>
                        <el-button size="mini" icon="el-icon-download" @click="download(scope.row)">下载</el-button>
                        <el-button size="mini" type="warning" icon="el-icon-refresh-left" @click="openRestoreDialog(scope.row)">
                            恢复
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <el-dialog title="创建加密冷备份" :visible.sync="coldDialog" width="460px">
            <el-alert
                class="dialog-alert"
                type="info"
                show-icon
                :closable="false"
                title="冷备份会导出加密后的数据库文件，恢复时必须输入同一个密码。"
            />
            <el-form label-width="96px">
                <el-form-item label="备份密码" required>
                    <el-input v-model="coldPassword" type="password" show-password autocomplete="new-password" />
                </el-form-item>
                <el-form-item label="确认密码" required>
                    <el-input v-model="coldPasswordConfirm" type="password" show-password autocomplete="new-password" />
                </el-form-item>
            </el-form>
            <span slot="footer">
                <el-button @click="coldDialog = false">取消</el-button>
                <el-button type="success" :loading="coldCreating" @click="createCold">创建冷备份</el-button>
            </span>
        </el-dialog>

        <el-dialog title="恢复备份" :visible.sync="restoreDialog" width="500px">
            <el-alert
                class="dialog-alert"
                type="warning"
                show-icon
                :closable="false"
                title="恢复会替换当前离线服务端数据库，系统会先自动创建一份恢复前快照。"
            />
            <el-descriptions v-if="selectedBackup" :column="1" border size="small">
                <el-descriptions-item label="备份编号">{{ selectedBackup.BackupNo }}</el-descriptions-item>
                <el-descriptions-item label="备份类型">
                    {{ selectedBackup.Encrypted ? '加密冷备份' : '热备份' }}
                </el-descriptions-item>
                <el-descriptions-item label="文件名">{{ selectedBackup.FileName }}</el-descriptions-item>
            </el-descriptions>
            <el-form class="restore-form" label-width="96px">
                <el-form-item v-if="selectedBackup && selectedBackup.Encrypted" label="恢复密码" required>
                    <el-input v-model="restorePassword" type="password" show-password autocomplete="current-password" />
                </el-form-item>
            </el-form>
            <span slot="footer">
                <el-button @click="restoreDialog = false">取消</el-button>
                <el-button type="warning" :loading="restoring" @click="confirmRestore">确认恢复</el-button>
            </span>
        </el-dialog>
    </div>
</template>

<script>
export default {
    name: 'BackupRestore',
    data() {
        return {
            loading: false,
            hotCreating: false,
            coldCreating: false,
            restoring: false,
            backups: [],
            keyword: '',
            coldDialog: false,
            restoreDialog: false,
            coldPassword: '',
            coldPasswordConfirm: '',
            restorePassword: '',
            selectedBackup: null
        }
    },
    computed: {
        filteredBackups() {
            const word = this.keyword.trim().toLowerCase()
            if (!word) return this.backups
            return this.backups.filter(item => {
                return String(item.BackupNo || '').toLowerCase().includes(word) ||
                    String(item.FileName || '').toLowerCase().includes(word)
            })
        },
        hotCount() {
            return this.backups.filter(item => !item.Encrypted).length
        },
        coldCount() {
            return this.backups.filter(item => item.Encrypted).length
        },
        restoredCount() {
            return this.backups.filter(item => item.Status === 'restored').length
        }
    },
    created() {
        this.fetchBackups()
    },
    methods: {
        async fetchBackups() {
            this.loading = true
            try {
                const response = await this.$offlineApi.listBackups()
                this.backups = response.data.data || []
            } catch (error) {
                this.$message.error(this.apiError(error, '查询备份列表失败'))
            } finally {
                this.loading = false
            }
        },

        async createHot() {
            this.hotCreating = true
            try {
                await this.$offlineApi.createHotBackup()
                this.$message.success('热备份已创建')
                await this.fetchBackups()
            } catch (error) {
                this.$message.error(this.apiError(error, '热备份创建失败'))
            } finally {
                this.hotCreating = false
            }
        },

        openColdDialog() {
            this.coldPassword = ''
            this.coldPasswordConfirm = ''
            this.coldDialog = true
        },

        async createCold() {
            if (!this.coldPassword || !this.coldPasswordConfirm) {
                this.$message.warning('请输入并确认冷备份密码')
                return
            }
            if (this.coldPassword !== this.coldPasswordConfirm) {
                this.$message.warning('两次输入的密码不一致')
                return
            }
            this.coldCreating = true
            try {
                await this.$offlineApi.createColdBackup(this.coldPassword)
                this.$message.success('加密冷备份已创建')
                this.coldDialog = false
                await this.fetchBackups()
            } catch (error) {
                this.$message.error(this.apiError(error, '冷备份创建失败'))
            } finally {
                this.coldCreating = false
            }
        },

        async verify(row) {
            try {
                const response = await this.$offlineApi.verifyBackup(row.ID)
                const data = response.data.data || {}
                if (data.valid) {
                    this.$message.success('备份文件校验通过')
                } else {
                    this.$alert(`当前哈希：${data.fileHash || '-'}`, '备份文件校验失败', { type: 'error' })
                }
            } catch (error) {
                this.$message.error(this.apiError(error, '备份校验失败'))
            }
        },

        async download(row) {
            try {
                const response = await this.$offlineApi.downloadBackupRecord(row.ID)
                const blob = new Blob([response.data])
                const url = window.URL.createObjectURL(blob)
                const link = document.createElement('a')
                link.href = url
                link.download = row.FileName || `${row.BackupNo}.backup`
                link.click()
                window.URL.revokeObjectURL(url)
            } catch (error) {
                this.$message.error(this.apiError(error, '备份下载失败'))
            }
        },

        openRestoreDialog(row) {
            this.selectedBackup = row
            this.restorePassword = ''
            this.restoreDialog = true
        },

        async confirmRestore() {
            if (!this.selectedBackup) return
            if (this.selectedBackup.Encrypted && !this.restorePassword) {
                this.$message.warning('请输入冷备份恢复密码')
                return
            }
            try {
                await this.$confirm(
                    `确认恢复备份 ${this.selectedBackup.BackupNo}？恢复后当前数据库会被替换。`,
                    '恢复确认',
                    { type: 'warning', confirmButtonText: '确认恢复', cancelButtonText: '取消' }
                )
            } catch {
                return
            }
            this.restoring = true
            try {
                await this.$offlineApi.restoreBackup(this.selectedBackup.ID, this.restorePassword)
                this.$message.success('备份已恢复')
                this.restoreDialog = false
                await this.fetchBackups()
            } catch (error) {
                this.$message.error(this.apiError(error, '备份恢复失败'))
            } finally {
                this.restoring = false
            }
        },

        statusType(status) {
            if (status === 'restored') return 'warning'
            if (status === 'created') return 'success'
            return 'info'
        },

        statusText(status) {
            const map = { created: '已创建', restored: '已恢复', failed: '失败' }
            return map[status] || status || '-'
        },

        formatTime(value) {
            if (!value) return '-'
            const date = typeof value === 'number' ? new Date(value * 1000) : new Date(value)
            if (Number.isNaN(date.getTime())) return '-'
            return date.toLocaleString()
        },

        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.toolbar {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    align-items: flex-start;
    margin-bottom: 16px;
}

.toolbar h2 {
    margin: 0 0 6px;
    font-size: 22px;
    font-weight: 600;
    color: #1f2937;
}

.toolbar p {
    margin: 0;
    color: #667085;
    font-size: 13px;
}

.actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: flex-end;
}

.summary-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    margin-bottom: 16px;
}

.summary-item {
    background: #fff;
    border: 1px solid #ebeef5;
    border-radius: 4px;
    padding: 14px 16px;
}

.summary-label {
    display: block;
    color: #667085;
    font-size: 13px;
    margin-bottom: 8px;
}

.summary-item strong {
    color: #1f2937;
    font-size: 24px;
    line-height: 1;
}

.table-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.search-input {
    width: 260px;
}

.dialog-alert,
.restore-form {
    margin-bottom: 14px;
}

@media (max-width: 960px) {
    .toolbar {
        flex-direction: column;
    }

    .summary-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
}
</style>
