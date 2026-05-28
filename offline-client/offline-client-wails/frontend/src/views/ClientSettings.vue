<template>
    <div class="client-settings-container">
        <el-card>
            <div slot="header" class="clearfix">
                <span>客户端设置</span>
            </div>

            <el-form label-width="140px" :model="form">
                <el-form-item label="服务器 HTTP 地址">
                    <el-input v-model="form.serverHttpUrl" @blur="syncWsUrl"></el-input>
                </el-form-item>

                <el-form-item label="WebSocket 地址">
                    <el-input v-model="form.serverWsUrl"></el-input>
                </el-form-item>

                <el-form-item label="读卡器名称">
                    <el-input v-model="form.cardReaderName" placeholder="留空则自动选择可用读卡器"></el-input>
                </el-form-item>

                <el-form-item label="当前用户">
                    <el-input :value="currentUsername" disabled></el-input>
                </el-form-item>

                <el-form-item label="Token">
                    <el-input :value="maskedToken" disabled></el-input>
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" :loading="saving" @click="save">保存</el-button>
                    <el-button :loading="testingSe" @click="testSe">检测 SE</el-button>
                    <el-button @click="reconnectWs">重连 WebSocket</el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { deriveWsUrl } from '../services/settings'
import { seApi } from '../services/wails-api'

export default {
    name: 'ClientSettings',
    data() {
        return {
            form: {
                ...this.$store.state.clientSettings
            },
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
        syncWsUrl() {
            this.form.serverWsUrl = deriveWsUrl(this.form.serverHttpUrl)
        },
        async save() {
            this.saving = true
            try {
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
                this.$alert(response.data.cplc_info || '', 'SE CPLC', {
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
</style>
