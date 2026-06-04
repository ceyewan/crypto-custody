<template>
    <div class="page offline-tasks-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">离线任务</h2>
                <p class="page-subtitle">通过在线系统导出的 JSON 任务包生成托管地址、完成交易签名，并下载 JSON 结果包。</p>
            </div>
            <el-button icon="el-icon-refresh" @click="resetFlow">重新选择</el-button>
        </div>

        <el-card>
            <el-steps :active="stepActive" finish-status="success" simple class="task-steps">
                <el-step title="导入任务包"></el-step>
                <el-step title="选择参与方"></el-step>
                <el-step title="发起协同处理"></el-step>
                <el-step title="导出结果包"></el-step>
            </el-steps>

            <el-tabs v-model="activeTab">
                <el-tab-pane v-if="isAdmin" label="导入 JSON" name="import">
                    <div class="section-grid">
                        <div>
                            <h3>选择任务包</h3>
                            <p class="muted">支持在线系统导出的任务包，导入时会校验包方向、任务类型和内容摘要。</p>
                            <input ref="taskFile" type="file" accept="application/json,.json" @change="handleFileChange">
                            <div class="file-name" v-if="fileName">{{ fileName }}</div>
                            <div class="actions">
                                <el-button type="primary" icon="el-icon-upload2" :loading="importing" @click="importTask">
                                    导入任务包
                                </el-button>
                            </div>
                        </div>

                        <div>
                            <h3>任务摘要</h3>
                            <el-empty v-if="!taskPackage && !importedTask" description="尚未选择任务包" :image-size="72"></el-empty>
                            <el-descriptions v-else :column="1" border size="small">
                                <el-descriptions-item label="任务编号">{{ currentTaskNo }}</el-descriptions-item>
                                <el-descriptions-item label="任务类型">{{ taskTypeText }}</el-descriptions-item>
                                <el-descriptions-item label="关联案件编号">{{ payload.case_no || '-' }}</el-descriptions-item>
                                <el-descriptions-item v-if="isSignTask" label="关联交易编号">{{ payload.transaction_no || '-' }}</el-descriptions-item>
                                <el-descriptions-item v-if="isSignTask" label="签名地址">{{ payload.from_address || '-' }}</el-descriptions-item>
                                <el-descriptions-item v-if="isSignTask" label="消息哈希">{{ payload.message_hash || '-' }}</el-descriptions-item>
                                <el-descriptions-item v-if="isKeygenTask" label="门限策略">
                                    {{ requiredSigners || '-' }} / {{ totalParties || '-' }}
                                </el-descriptions-item>
                                <el-descriptions-item label="内容摘要">{{ currentPayloadHash || '-' }}</el-descriptions-item>
                                <el-descriptions-item v-if="importedTask" label="导入状态">
                                    <el-tag type="success">{{ importedTask.status || 'imported' }}</el-tag>
                                </el-descriptions-item>
                            </el-descriptions>
                        </div>
                    </div>
                </el-tab-pane>

                <el-tab-pane label="任务记录" name="records">
                    <el-table :data="tasks" v-loading="loadingTasks" style="width: 100%">
                        <el-table-column prop="task_no" label="任务编号" min-width="170"></el-table-column>
                        <el-table-column prop="task_type" label="类型" width="150">
                            <template slot-scope="scope">
                                {{ taskTypeLabel(scope.row.task_type) }}
                            </template>
                        </el-table-column>
                        <el-table-column prop="payload_hash" label="任务内容摘要" min-width="220"></el-table-column>
                        <el-table-column prop="result_hash" label="结果内容摘要" min-width="220"></el-table-column>
                        <el-table-column prop="status" label="状态" width="120">
                            <template slot-scope="scope">
                                <el-tag :type="taskStatusTag(scope.row.status)">{{ taskStatusText(scope.row.status) }}</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="updated_at" label="更新时间" width="170">
                            <template slot-scope="scope">
                                {{ formatTime(scope.row.updated_at) }}
                            </template>
                        </el-table-column>
                        <el-table-column label="操作" width="180">
                            <template slot-scope="scope">
                                <el-button size="mini" @click="loadTaskRecord(scope.row)">加载</el-button>
                                <el-button size="mini" type="primary" @click="prepareDownload(scope.row)">结果</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-empty v-if="!loadingTasks && tasks.length === 0" description="暂无离线任务记录" :image-size="90"></el-empty>
                </el-tab-pane>

                <el-tab-pane v-if="isAdmin" label="发起任务" name="start">
                    <el-alert
                        v-if="!currentTaskNo"
                        type="warning"
                        :closable="false"
                        title="请先导入在线系统导出的 JSON 任务包。">
                    </el-alert>

                    <div v-else class="section-grid">
                        <div>
                            <h3>参与方</h3>
                            <p class="muted">
                                管理员既可以发起任务，也可以作为参与方；警员可以参与；审计员只查看，不会出现在候选列表中。
                            </p>
                            <el-select
                                v-model="selectedParticipants"
                                multiple
                                filterable
                                placeholder="请选择参与方"
                                style="width: 100%">
                                <el-option
                                    v-for="user in participantOptions"
                                    :key="user.username"
                                    :label="participantLabel(user)"
                                    :value="user.username">
                                </el-option>
                            </el-select>

                            <div class="actions">
                                <el-button icon="el-icon-refresh" :loading="loadingParticipants" @click="loadParticipants">
                                    刷新候选人
                                </el-button>
                            </div>

                            <el-descriptions :column="1" border size="small" class="summary">
                                <el-descriptions-item label="任务编号">{{ currentTaskNo }}</el-descriptions-item>
                                <el-descriptions-item label="任务类型">{{ taskTypeText }}</el-descriptions-item>
                                <el-descriptions-item v-if="isKeygenTask" label="门限策略">
                                    {{ requiredSigners }} / {{ totalParties }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="isSignTask" label="签名地址">{{ payload.from_address }}</el-descriptions-item>
                                <el-descriptions-item v-if="isSignTask" label="私钥门限">
                                    {{ keyThresholdText }}
                                </el-descriptions-item>
                                <el-descriptions-item label="已选参与方">{{ selectedParticipants.join(', ') || '-' }}</el-descriptions-item>
                            </el-descriptions>
                        </div>

                        <div>
                            <h3>发起参数</h3>
                            <el-form label-width="120px">
                                <el-form-item label="私钥编号">
                                    <el-input v-model="offlineKeyID" :placeholder="isKeygenTask ? '默认 OFFKEY-任务编号' : '签名任务可留空由地址匹配'"></el-input>
                                </el-form-item>
                                <el-form-item>
                                    <el-button type="primary" icon="el-icon-s-promotion" :loading="starting" @click="startTask">
                                        发起 {{ taskTypeText }}
                                    </el-button>
                                </el-form-item>
                            </el-form>

                            <el-alert
                                v-if="startMessage"
                                type="success"
                                :closable="false"
                                title="邀请已发送，等待参与方确认并完成协同处理。">
                            </el-alert>
                            <el-input
                                v-if="startMessage"
                                :value="JSON.stringify(startMessage, null, 2)"
                                type="textarea"
                                :rows="8"
                                readonly
                                class="message-preview">
                            </el-input>
                        </div>
                    </div>
                </el-tab-pane>

                <el-tab-pane v-if="isAdmin" label="下载结果 JSON" name="result">
                    <div class="section-grid">
                        <div>
                            <h3>结果包</h3>
                            <p class="muted">托管地址和私钥生成完成后下载 custody_keygen_result，签名完成后下载 sign_result。</p>
                            <el-form label-width="100px">
                                <el-form-item label="任务编号">
                                    <el-input v-model="resultTaskNo" placeholder="例如 TASK-20260604-001"></el-input>
                                </el-form-item>
                                <el-form-item>
                                    <el-button type="primary" icon="el-icon-download" :loading="downloading" @click="downloadResult">
                                        下载 JSON 结果包
                                    </el-button>
                                </el-form-item>
                            </el-form>
                        </div>

                        <div>
                            <h3>结果摘要</h3>
                            <el-empty v-if="!resultPackage" description="尚未下载结果包" :image-size="72"></el-empty>
                            <el-descriptions v-else :column="1" border size="small">
                                <el-descriptions-item label="任务编号">{{ resultPackage.task_no }}</el-descriptions-item>
                                <el-descriptions-item label="结果类型">{{ taskTypeLabel(resultPackage.task_type) }}</el-descriptions-item>
                                <el-descriptions-item label="内容摘要">{{ resultPackage.payload_hash }}</el-descriptions-item>
                                <el-descriptions-item v-if="resultPayload.custody_address" label="托管地址">
                                    {{ resultPayload.custody_address }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="resultPayload.public_key" label="公钥">
                                    {{ resultPayload.public_key }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="resultPayload.signature" label="签名">
                                    {{ resultPayload.signature }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="resultPayload.offline_ref_no" label="离线引用号">
                                    {{ resultPayload.offline_ref_no }}
                                </el-descriptions-item>
                            </el-descriptions>
                        </div>
                    </div>
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
            fileName: '',
            taskPackage: null,
            importedTask: null,
            importing: false,
            loadingParticipants: false,
            starting: false,
            downloading: false,
            participantOptions: [],
            selectedParticipants: [],
            offlineKeyID: '',
            keyInfo: null,
            startMessage: null,
            resultTaskNo: '',
            resultPackage: null,
            tasks: [],
            loadingTasks: false
        }
    },
    computed: {
        payload() {
            return this.taskPackage?.payload || {}
        },
        currentTaskNo() {
            return this.importedTask?.task_no || this.taskPackage?.task_no || ''
        },
        currentTaskType() {
            return this.importedTask?.task_type || this.taskPackage?.task_type || ''
        },
        currentPayloadHash() {
            return this.importedTask?.payload_hash || this.taskPackage?.payload_hash || ''
        },
        isKeygenTask() {
            return this.currentTaskType === 'custody_keygen'
        },
        isSignTask() {
            return this.currentTaskType === 'sign'
        },
        taskTypeText() {
            if (this.isKeygenTask) return '生成托管地址和私钥'
            if (this.isSignTask) return '交易签名'
            return this.currentTaskType || '-'
        },
        requiredSigners() {
            return this.payload.threshold_policy?.required_signers || ''
        },
        totalParties() {
            return this.payload.threshold_policy?.total_parties || ''
        },
        stepActive() {
            if (this.resultPackage) return 4
            if (this.startMessage) return 3
            if (this.selectedParticipants.length) return 2
            if (this.importedTask) return 1
            return 0
        },
        resultPayload() {
            return this.resultPackage?.payload || {}
        },
        keyThresholdText() {
            if (!this.keyInfo) {
                return '由服务端校验'
            }
            return `${this.keyInfo.required_signers} / ${this.keyInfo.total_parties}`
        },
        isAdmin() {
            return this.$store.getters.isAdmin
        }
    },
    created() {
        if (!this.isAdmin) {
            this.activeTab = 'records'
        }
        this.loadTaskList()
    },
    methods: {
        async loadTaskList() {
            this.loadingTasks = true
            try {
                const response = await this.$offlineApi.listTasks()
                this.tasks = response.data.tasks || []
            } catch (error) {
                this.$message.error(this.apiError(error, '加载任务记录失败'))
            } finally {
                this.loadingTasks = false
            }
        },

        resetFlow() {
            this.fileName = ''
            this.taskPackage = null
            this.importedTask = null
            this.participantOptions = []
            this.selectedParticipants = []
            this.offlineKeyID = ''
            this.keyInfo = null
            this.startMessage = null
            this.resultPackage = null
            this.resultTaskNo = ''
            if (this.$refs.taskFile) {
                this.$refs.taskFile.value = ''
            }
            this.activeTab = 'import'
        },

        handleFileChange(event) {
            const file = event.target.files && event.target.files[0]
            if (!file) {
                this.fileName = ''
                this.taskPackage = null
                return
            }
            this.fileName = file.name
            const reader = new FileReader()
            reader.onload = () => {
                try {
                    const parsed = JSON.parse(reader.result)
                    this.validateTaskPackage(parsed)
                    this.taskPackage = parsed
                    this.importedTask = null
                    this.resultPackage = null
                    this.startMessage = null
                    this.selectedParticipants = []
                    this.offlineKeyID = parsed.task_type === 'custody_keygen' ? `OFFKEY-${parsed.task_no}` : ''
                    this.resultTaskNo = parsed.task_no || ''
                    this.$message.success('任务包已解析，请确认摘要后导入')
                } catch (error) {
                    this.taskPackage = null
                    this.$message.error('任务包 JSON 无效: ' + error.message)
                }
            }
            reader.readAsText(file)
        },

        validateTaskPackage(pkg) {
            if (pkg.schema_version !== '1.0') throw new Error('schema_version 必须为 1.0')
            if (pkg.package_type !== 'offline_task') throw new Error('package_type 必须为 offline_task')
            if (!['custody_keygen', 'sign'].includes(pkg.task_type)) throw new Error('仅支持 custody_keygen/sign')
            if (!pkg.task_no) throw new Error('缺少 task_no')
            if (!pkg.payload || typeof pkg.payload !== 'object') throw new Error('缺少 payload')
            if (!pkg.payload_hash) throw new Error('缺少 payload_hash')
        },

        async importTask() {
            if (!this.taskPackage) {
                this.$message.warning('请选择 JSON 任务包')
                return
            }
            this.importing = true
            try {
                const response = await this.$offlineApi.importTask(this.taskPackage)
                this.importedTask = response.data.task
                this.resultTaskNo = this.importedTask.task_no
                this.$message.success(response.data.duplicated ? '任务包已存在，内容一致' : '任务包已导入')
                await this.loadTaskList()
                await this.loadParticipants()
                this.activeTab = 'start'
            } catch (error) {
                this.$message.error(this.apiError(error, '导入失败'))
            } finally {
                this.importing = false
            }
        },

        async loadTaskRecord(task) {
            try {
                const response = await this.$offlineApi.getTask(task.task_no)
                this.importedTask = response.data.task
                this.taskPackage = {
                    schema_version: '1.0',
                    package_type: 'offline_task',
                    task_type: response.data.task.task_type,
                    task_no: response.data.task.task_no,
                    payload: response.data.payload || {},
                    payload_hash: response.data.task.payload_hash
                }
                this.resultTaskNo = response.data.task.task_no
                this.offlineKeyID = response.data.task.task_type === 'custody_keygen' ? `OFFKEY-${response.data.task.task_no}` : ''
                this.resultPackage = null
                this.startMessage = null
                await this.loadParticipants()
                this.activeTab = 'start'
            } catch (error) {
                this.$message.error(this.apiError(error, '加载任务失败'))
            }
        },

        prepareDownload(task) {
            this.resultTaskNo = task.task_no
            this.activeTab = 'result'
        },

        async loadParticipants() {
            if (!this.currentTaskType) {
                return
            }
            this.loadingParticipants = true
            this.keyInfo = null
            try {
                let users = []
                if (this.isSignTask) {
                    const address = this.payload.from_address
                    if (!address) {
                        throw new Error('签名任务缺少 from_address')
                    }
                    const response = await this.$signApi.getAvailableUsers(address)
                    users = response.data.data || response.data.users || []
                    try {
                        const keyResponse = await this.$offlineApi.getKey(address)
                        this.keyInfo = keyResponse.data.key
                    } catch {
                        this.keyInfo = null
                    }
                } else {
                    const response = await this.$keygenApi.getAvailableUsers()
                    users = response.data.data || response.data.users || []
                }

                this.participantOptions = users.filter(user => ['admin', 'officer'].includes(user.role))
                this.autoSelectParticipants()
            } catch (error) {
                this.participantOptions = []
                this.selectedParticipants = []
                this.$message.error(this.apiError(error, '加载参与方失败'))
            } finally {
                this.loadingParticipants = false
            }
        },

        autoSelectParticipants() {
            const names = this.participantOptions.map(user => user.username)
            if (this.isKeygenTask && this.totalParties) {
                this.selectedParticipants = names.slice(0, Number(this.totalParties))
                return
            }
            if (this.isSignTask) {
                const required = this.keyInfo?.required_signers || 2
                this.selectedParticipants = names.slice(0, Math.min(required, names.length))
            }
        },

        participantLabel(user) {
            const roleText = user.role === 'admin' ? '管理员' : '警员'
            const name = user.nickname || user.username
            return `${name} (${user.username}, ${roleText})`
        },

        async startTask() {
            if (!this.importedTask) {
                this.$message.warning('请先导入任务包')
                return
            }
            if (!this.selectedParticipants.length) {
                this.$message.warning('请选择参与方')
                return
            }
            if (this.isKeygenTask && Number(this.totalParties) !== this.selectedParticipants.length) {
                this.$message.warning(`keygen 需要选择 ${this.totalParties} 个参与方`)
                return
            }

            this.starting = true
            try {
                const payload = {
                    participants: this.selectedParticipants,
                    offline_key_id: this.offlineKeyID
                }
                const response = this.isSignTask
                    ? await this.$offlineApi.buildSignRequest(this.currentTaskNo, payload)
                    : await this.$offlineApi.buildKeygenRequest(this.currentTaskNo, payload)
                this.startMessage = response.data.message
                if (!sendWSMessage(this.startMessage)) {
                    throw new Error('服务连接未建立')
                }
                this.$store.commit('setCurrentSession', this.startMessage.session_key)
                this.$message.success('邀请已发送')
                this.activeTab = 'result'
            } catch (error) {
                this.$message.error(this.apiError(error, '发起失败'))
            } finally {
                this.starting = false
            }
        },

        async downloadResult() {
            const taskNo = (this.resultTaskNo || this.currentTaskNo || '').trim()
            if (!taskNo) {
                this.$message.warning('请输入任务编号')
                return
            }
            this.downloading = true
            try {
                const response = await this.$offlineApi.downloadResult(taskNo)
                this.resultPackage = response.data
                const content = JSON.stringify(response.data, null, 2)
                const fileName = `offline_result_${this.sanitizeFilePart(taskNo)}.json`
                const savedPath = await this.$fileApi.saveJSON(fileName, content)
                if (savedPath) {
                    this.$message.success(`JSON 结果包已保存：${savedPath}`)
                } else {
                    this.$message.info('已取消保存')
                }
            } catch (error) {
                this.$message.error(this.apiError(error, '下载失败'))
            } finally {
                this.downloading = false
            }
        },

        sanitizeFilePart(value) {
            return value.replace(/[^a-zA-Z0-9_-]/g, '_') || 'task'
        },

        taskTypeLabel(type) {
            if (type === 'custody_keygen') return '生成托管地址和私钥'
            if (type === 'sign') return '交易签名'
            if (type === 'custody_keygen_result') return '托管地址生成结果'
            if (type === 'sign_result') return '交易签名结果'
            return type
        },

        taskStatusTag(status) {
            if (status === 'completed') return 'success'
            if (status === 'failed') return 'danger'
            if (status === 'processing') return 'warning'
            return 'info'
        },

        taskStatusText(status) {
            const map = {
                imported: '已导入',
                processing: '处理中',
                completed: '已完成',
                failed: '失败'
            }
            return map[status] || status || '-'
        },

        formatTime(value) {
            return value ? new Date(value).toLocaleString() : '-'
        },

        apiError(error, fallback) {
            return error.response?.data?.error || error.response?.data?.message || error.message || fallback
        }
    }
}
</script>

<style scoped>
.offline-tasks-page {
    min-height: calc(100vh - 60px);
}

.task-steps {
    margin-bottom: 18px;
}

.section-grid {
    display: grid;
    grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
    gap: 24px;
}

h3 {
    margin: 0 0 8px;
    font-size: 16px;
}

.muted {
    margin: 0 0 16px;
    color: #606266;
    font-size: 13px;
    line-height: 1.6;
}

.file-name {
    margin-top: 10px;
    color: #409EFF;
    font-size: 13px;
}

.actions {
    margin-top: 14px;
}

.summary {
    margin-top: 18px;
}

.message-preview {
    margin-top: 14px;
}

@media (max-width: 900px) {
    .section-grid {
        grid-template-columns: 1fr;
    }
}
</style>
