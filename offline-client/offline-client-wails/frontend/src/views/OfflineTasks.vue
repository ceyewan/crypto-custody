<template>
    <div class="offline-tasks-container">
        <el-card>
            <div slot="header">
                <span>离线任务</span>
            </div>

            <el-tabs v-model="activeTab">
                <el-tab-pane label="导入" name="import">
                    <el-form label-width="120px">
                        <el-form-item label="任务包">
                            <input ref="taskFile" type="file" accept="application/json,.json" @change="handleFileChange">
                        </el-form-item>
                        <el-form-item>
                            <el-button type="primary" :loading="importing" @click="importTask">
                                导入任务包
                            </el-button>
                        </el-form-item>
                    </el-form>

                    <el-alert v-if="importedTask" type="success" :closable="false">
                        <span>已导入: {{ importedTask.task_no }} / {{ importedTask.task_type }}</span>
                    </el-alert>
                </el-tab-pane>

                <el-tab-pane label="发起" name="start">
                    <el-form :model="startForm" label-width="120px">
                        <el-form-item label="任务编号">
                            <el-input v-model="startForm.taskNo"></el-input>
                        </el-form-item>
                        <el-form-item label="任务类型">
                            <el-select v-model="startForm.taskType" style="width: 100%">
                                <el-option label="托管钱包生成" value="custody_keygen"></el-option>
                                <el-option label="交易签名" value="sign"></el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item label="参与者">
                            <el-select v-model="startForm.participants" multiple filterable allow-create style="width: 100%">
                                <el-option v-for="name in knownParticipants" :key="name" :label="name" :value="name"></el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item label="离线密钥ID">
                            <el-input v-model="startForm.offlineKeyID"></el-input>
                        </el-form-item>
                        <el-form-item>
                            <el-button type="primary" :loading="starting" @click="startTask">
                                发送请求
                            </el-button>
                            <el-button @click="loadTask">查询任务</el-button>
                        </el-form-item>
                    </el-form>

                    <el-input
                        v-if="lastMessage"
                        :value="JSON.stringify(lastMessage, null, 2)"
                        type="textarea"
                        :rows="10"
                        readonly>
                    </el-input>
                </el-tab-pane>

                <el-tab-pane label="结果" name="result">
                    <el-form label-width="120px">
                        <el-form-item label="任务编号">
                            <el-input v-model="resultTaskNo"></el-input>
                        </el-form-item>
                        <el-form-item>
                            <el-button type="primary" :loading="downloading" @click="downloadResult">
                                下载结果包
                            </el-button>
                        </el-form-item>
                    </el-form>
                </el-tab-pane>
            </el-tabs>
        </el-card>
    </div>
</template>

<script>
import { sendWSMessage } from '../services/ws'

export default {
    name: 'OfflineTasks',
    data() {
        return {
            activeTab: 'import',
            taskPackage: null,
            importedTask: null,
            importing: false,
            starting: false,
            downloading: false,
            knownParticipants: [],
            startForm: {
                taskNo: '',
                taskType: 'custody_keygen',
                participants: [],
                offlineKeyID: ''
            },
            resultTaskNo: '',
            lastMessage: null
        }
    },
    created() {
        this.loadParticipants()
    },
    methods: {
        async loadParticipants() {
            try {
                const response = await this.$keygenApi.getAvailableUsers()
                this.knownParticipants = response.data.data
                    .filter(user => user.role === 'participant')
                    .map(user => user.username)
            } catch {
                this.knownParticipants = []
            }
        },

        handleFileChange(event) {
            const file = event.target.files && event.target.files[0]
            if (!file) {
                this.taskPackage = null
                return
            }
            const reader = new FileReader()
            reader.onload = () => {
                try {
                    this.taskPackage = JSON.parse(reader.result)
                    this.startForm.taskNo = this.taskPackage.task_no || ''
                    this.resultTaskNo = this.taskPackage.task_no || ''
                    if (this.taskPackage.task_type) {
                        this.startForm.taskType = this.taskPackage.task_type
                    }
                } catch (error) {
                    this.taskPackage = null
                    this.$message.error('任务包JSON格式错误: ' + error.message)
                }
            }
            reader.readAsText(file)
        },

        async importTask() {
            if (!this.taskPackage) {
                this.$message.warning('请选择任务包')
                return
            }
            this.importing = true
            try {
                const response = await this.$offlineApi.importTask(this.taskPackage)
                this.importedTask = response.data.task
                this.startForm.taskNo = this.importedTask.task_no
                this.resultTaskNo = this.importedTask.task_no
                this.startForm.taskType = this.importedTask.task_type
                this.$message.success('任务包已导入')
            } catch (error) {
                this.$message.error(this.apiError(error, '导入失败'))
            } finally {
                this.importing = false
            }
        },

        async loadTask() {
            if (!this.startForm.taskNo) {
                this.$message.warning('请输入任务编号')
                return
            }
            try {
                const response = await this.$offlineApi.getTask(this.startForm.taskNo)
                const task = response.data.task
                this.startForm.taskType = task.task_type
                this.resultTaskNo = task.task_no
                this.$message.success('任务已加载')
            } catch (error) {
                this.$message.error(this.apiError(error, '查询失败'))
            }
        },

        async startTask() {
            if (!this.startForm.taskNo) {
                this.$message.warning('请输入任务编号')
                return
            }
            if (!this.startForm.participants.length) {
                this.$message.warning('请选择参与者')
                return
            }
            this.starting = true
            try {
                const payload = {
                    participants: this.startForm.participants,
                    offline_key_id: this.startForm.offlineKeyID
                }
                const response = this.startForm.taskType === 'sign'
                    ? await this.$offlineApi.buildSignRequest(this.startForm.taskNo, payload)
                    : await this.$offlineApi.buildKeygenRequest(this.startForm.taskNo, payload)
                this.lastMessage = response.data.message
                if (!sendWSMessage(this.lastMessage)) {
                    throw new Error('WebSocket未连接')
                }
                this.$store.commit('setCurrentSession', this.lastMessage.session_key)
                this.$message.success('请求已发送')
                this.$router.push('/notifications')
            } catch (error) {
                this.$message.error(this.apiError(error, '发送失败'))
            } finally {
                this.starting = false
            }
        },

        async downloadResult() {
            if (!this.resultTaskNo) {
                this.$message.warning('请输入任务编号')
                return
            }
            this.downloading = true
            try {
                const response = await this.$offlineApi.downloadResult(this.resultTaskNo)
                const content = JSON.stringify(response.data, null, 2)
                const blob = new Blob([content], { type: 'application/json' })
                const url = URL.createObjectURL(blob)
                const link = document.createElement('a')
                link.href = url
                link.download = `offline_result_${this.resultTaskNo}.json`
                link.click()
                URL.revokeObjectURL(url)
                this.$message.success('结果包已生成')
            } catch (error) {
                this.$message.error(this.apiError(error, '下载失败'))
            } finally {
                this.downloading = false
            }
        },

        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.offline-tasks-container {
    padding: 20px;
}
</style>
