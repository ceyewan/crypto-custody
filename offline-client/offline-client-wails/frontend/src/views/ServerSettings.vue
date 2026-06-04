<template>
    <div class="login-container">
        <el-card class="settings-card">
            <div slot="header" class="card-header">
                <h2>服务器设置</h2>
                <p>通常只需要填写离线服务器 IP 或域名。</p>
            </div>

            <el-form label-width="0">
                <el-form-item>
                    <el-input
                        v-model="serverHost"
                        prefix-icon="el-icon-monitor"
                        placeholder="例如 192.168.1.20 或 offline-server.local"
                        @input="hostDirty = true"
                        @blur="syncFromHost">
                    </el-input>
                </el-form-item>

                <el-descriptions :column="1" border size="small" class="derived">
                    <el-descriptions-item label="HTTP 地址">{{ form.serverHttpUrl }}</el-descriptions-item>
                    <el-descriptions-item label="实时连接地址">{{ form.serverWsUrl }}</el-descriptions-item>
                </el-descriptions>

                <el-collapse class="advanced">
                    <el-collapse-item title="高级设置" name="advanced">
                        <el-form label-width="130px">
                            <el-form-item label="HTTP 地址">
                                <el-input v-model="form.serverHttpUrl" @blur="syncWsUrl"></el-input>
                            </el-form-item>
                            <el-form-item label="实时连接地址">
                                <el-input v-model="form.serverWsUrl"></el-input>
                            </el-form-item>
                            <el-form-item label="读卡器名称">
                                <el-input v-model="form.cardReaderName" placeholder="留空自动选择"></el-input>
                            </el-form-item>
                        </el-form>
                    </el-collapse-item>
                </el-collapse>

                <el-form-item class="actions">
                    <el-button type="primary" :loading="saving" @click="save">保存</el-button>
                    <el-button :loading="testing" @click="testConnection">检查连接</el-button>
                    <el-button @click="$router.push('/login')">返回登录</el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import axios from 'axios'
import { deriveWsUrl, normalizeHttpUrl } from '../services/settings'

export default {
    name: 'ServerSettings',
    data() {
        const settings = this.$store.state.clientSettings
        return {
            serverHost: this.hostFromHttp(settings.serverHttpUrl),
            form: { ...settings },
            hostDirty: false,
            saving: false,
            testing: false
        }
    },
    methods: {
        hostFromHttp(value) {
            try {
                const url = new URL(value)
                return url.hostname
            } catch {
                return ''
            }
        },
        syncFromHost() {
            const host = (this.serverHost || '').trim()
            if (!host) {
                return
            }
            this.form.serverHttpUrl = normalizeHttpUrl(host)
            this.form.serverWsUrl = deriveWsUrl(this.form.serverHttpUrl)
            this.hostDirty = false
        },
        syncWsUrl() {
            this.form.serverHttpUrl = normalizeHttpUrl(this.form.serverHttpUrl)
            this.form.serverWsUrl = deriveWsUrl(this.form.serverHttpUrl)
            this.serverHost = this.hostFromHttp(this.form.serverHttpUrl)
            this.hostDirty = false
        },
        normalizeFormForSave() {
            if (this.hostDirty) {
                this.syncFromHost()
                return
            }
            this.form.serverHttpUrl = normalizeHttpUrl(this.form.serverHttpUrl || this.serverHost)
            if (!this.form.serverWsUrl) {
                this.form.serverWsUrl = deriveWsUrl(this.form.serverHttpUrl)
            }
            this.serverHost = this.hostFromHttp(this.form.serverHttpUrl)
        },
        async save() {
            this.saving = true
            try {
                this.normalizeFormForSave()
                const saved = await this.$store.dispatch('saveClientSettings', this.form)
                this.form = { ...saved }
                this.$message.success('服务器设置已保存')
                this.$router.push('/login')
            } catch (error) {
                this.$message.error('保存失败: ' + error.message)
            } finally {
                this.saving = false
            }
        },
        async testConnection() {
            this.testing = true
            try {
                this.normalizeFormForSave()
                await axios.get(`${this.form.serverHttpUrl}/__health_probe__`, {
                    timeout: 3000,
                    validateStatus: status => status < 500
                })
                await this.testWebSocket()
                this.$message.success('服务连接可达')
            } catch (error) {
                this.$message.error('连接失败: ' + (error.message || '请检查 IP 和端口'))
            } finally {
                this.testing = false
            }
        },
        testWebSocket() {
            return new Promise((resolve, reject) => {
                let settled = false
                const ws = new WebSocket(this.form.serverWsUrl)
                const timer = setTimeout(() => {
                    if (!settled) {
                        settled = true
                        ws.close()
                        reject(new Error('实时连接超时'))
                    }
                }, 3000)
                ws.onopen = () => {
                    if (!settled) {
                        settled = true
                        clearTimeout(timer)
                        ws.close(1000, 'settings-test')
                        resolve()
                    }
                }
                ws.onerror = () => {
                    if (!settled) {
                        settled = true
                        clearTimeout(timer)
                        reject(new Error('实时连接不可达'))
                    }
                }
            })
        }
    }
}
</script>

<style scoped>
.login-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #f0f2f5;
}

.settings-card {
    width: 520px;
    border-radius: 8px;
}

.card-header {
    text-align: center;
}

.card-header h2 {
    margin: 0;
    color: #304156;
}

.card-header p {
    margin: 8px 0 0;
    color: #606266;
    font-size: 13px;
}

.derived {
    margin-bottom: 16px;
}

.advanced {
    margin-bottom: 18px;
}

.actions {
    margin-bottom: 0;
}
</style>
