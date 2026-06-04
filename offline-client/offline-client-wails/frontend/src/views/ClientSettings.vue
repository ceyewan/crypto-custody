<template>
    <div class="client-settings-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>客户端设置</span>
            </div>

            <el-form label-width="140px" :model="form">
                <el-form-item label="服务器 IP/域名">
                    <el-input
                        v-model="serverHost"
                        placeholder="例如 192.168.1.20"
                        @input="hostDirty = true"
                        @blur="syncFromHost">
                    </el-input>
                </el-form-item>

                <el-descriptions :column="1" border size="small" class="derived">
                    <el-descriptions-item label="HTTP 地址">{{ form.serverHttpUrl }}</el-descriptions-item>
                    <el-descriptions-item label="实时连接地址">{{ form.serverWsUrl }}</el-descriptions-item>
                </el-descriptions>

                <el-form-item label="读卡器名称">
                    <el-input v-model="form.cardReaderName" placeholder="留空则自动选择可用读卡器"></el-input>
                </el-form-item>

                <el-collapse class="advanced">
                    <el-collapse-item title="高级连接设置" name="advanced">
                        <el-form-item label="服务器 HTTP 地址">
                            <el-input v-model="form.serverHttpUrl" @blur="syncWsUrl"></el-input>
                        </el-form-item>

                        <el-form-item label="实时连接地址">
                            <el-input v-model="form.serverWsUrl"></el-input>
                        </el-form-item>
                    </el-collapse-item>
                </el-collapse>

                <el-form-item label="当前用户">
                    <el-input :value="currentUsername" disabled></el-input>
                </el-form-item>

                <el-form-item label="Token">
                    <el-input :value="maskedToken" disabled></el-input>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" :loading="saving" @click="save">保存</el-button>
                    <el-button :loading="testingSe" @click="testSe">检测 SE</el-button>
                    <el-button @click="reconnectWs">重连服务连接</el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { deriveWsUrl, normalizeHttpUrl } from '../services/settings'
import { seApi } from '../services/wails-api'

export default {
    name: 'ClientSettings',
    data() {
        return {
            serverHost: this.hostFromHttp(this.$store.state.clientSettings.serverHttpUrl),
            form: {
                ...this.$store.state.clientSettings
            },
            hostDirty: false,
            saving: false,
            testingSe: false
        }
    },
    computed: {
        ...mapGetters(['currentUser']),
        currentUsername() {
            return this.currentUser ? this.currentUser.username : ''
        },
        maskedToken() {
            const token = this.$store.state.token || ''
            if (!token) {
                return ''
            }
            if (token.length <= 16) {
                return '********'
            }
            return `${token.slice(0, 8)}...${token.slice(-8)}`
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
                this.$message.success('客户端设置已保存')
            } catch (error) {
                this.$message.error('保存客户端设置失败: ' + error.message)
            } finally {
                this.saving = false
            }
        },
        async testSe() {
            this.testingSe = true
            try {
                await this.$store.dispatch('saveClientSettings', this.form)
                const response = await seApi.getCPLC()
                this.$alert(response.data.cplc_info || '', '安全芯片编号', {
                    closeOnClickModal: true
                })
            } catch (error) {
                this.$message.error('检测 SE 失败: ' + error.message)
            } finally {
                this.testingSe = false
            }
        },
        reconnectWs() {
            this.$store.dispatch('resetWebSocketConnection')
        }
    }
}
</script>

<style scoped>
.client-settings-container {
    padding: 20px;
}

.derived,
.advanced {
    margin-bottom: 18px;
}
</style>
