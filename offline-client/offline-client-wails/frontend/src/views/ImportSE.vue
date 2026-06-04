<template>
    <div class="page se-page">
        <el-card>
            <div slot="header" class="header">
                <span>SE 管理</span>
                <div>
                    <el-button size="small" type="primary" icon="el-icon-cpu" @click="openImportDialog">导入 SE</el-button>
                    <el-button size="small" icon="el-icon-refresh" :loading="loadingList" @click="loadSecurityElements">刷新</el-button>
                </div>
            </div>

            <el-form :inline="true" :model="query" class="query">
                <el-form-item><el-input v-model="query.seid" clearable placeholder="SEID" /></el-form-item>
                <el-form-item><el-input v-model="query.cplc" clearable placeholder="CPLC" /></el-form-item>
                <el-form-item>
                    <el-select v-model="query.status" clearable placeholder="状态">
                        <el-option label="active" value="active" />
                        <el-option label="disabled" value="disabled" />
                        <el-option label="lost" value="lost" />
                        <el-option label="destroyed" value="destroyed" />
                    </el-select>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" @click="applyQuery">查询</el-button>
                    <el-button @click="resetQuery">重置</el-button>
                </el-form-item>
            </el-form>

            <el-table :data="filteredSeList" v-loading="loadingList" empty-text="暂无 SE 记录">
                <el-table-column prop="se_id" label="SEID" width="170" show-overflow-tooltip />
                <el-table-column prop="cplc" label="CPLC" min-width="260" show-overflow-tooltip />
                <el-table-column label="状态" width="100">
                    <template slot-scope="scope">
                        <el-tag :type="scope.row.status === 'active' ? 'success' : 'warning'" size="small">{{ scope.row.status }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="registered_by" label="登记人" width="130" />
                <el-table-column prop="remark" label="备注/位置" min-width="160" show-overflow-tooltip />
                <el-table-column label="登记时间" width="170">
                    <template slot-scope="scope">{{ formatTime(scope.row.created_at) }}</template>
                </el-table-column>
                <el-table-column label="最近使用时间" width="170">
                    <template slot-scope="scope">{{ formatTime(scope.row.last_used_at) }}</template>
                </el-table-column>
                <el-table-column label="操作" width="100" fixed="right">
                    <template slot-scope="scope">
                        <el-button
                            type="text"
                            size="small"
                            icon="el-icon-delete"
                            :loading="deletingSeId === scope.row.se_id"
                            @click="deleteSe(scope.row)"
                        >删除</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <el-dialog title="导入 SE" :visible.sync="importDialogVisible" width="620px">
            <el-form :model="form" label-width="110px">
                <el-form-item label="安全芯片 ID">
                    <el-input v-model="form.seid" placeholder="系统自动建议，可按贴纸编号调整" />
                </el-form-item>
                <el-form-item label="CPLC">
                    <el-input v-model="form.cplc" type="textarea" :rows="3" readonly placeholder="点击读取当前 SE" />
                </el-form-item>
                <el-form-item label="保管位置">
                    <el-input v-model="form.custodyLocation" placeholder="例如 保险柜 A-01，可留空" />
                </el-form-item>
                <el-form-item>
                    <el-button icon="el-icon-cpu" :loading="reading" @click="readCurrentSe">读取当前 SE</el-button>
                </el-form-item>
            </el-form>
            <span slot="footer">
                <el-button @click="importDialogVisible = false">取消</el-button>
                <el-button type="primary" :loading="importing" @click="importCurrentSe">导入当前 SE</el-button>
            </span>
        </el-dialog>
    </div>
</template>

<script>
import { seApi as wailsSeApi } from '../services/wails-api'
import { seApi as serverSeApi } from '../services/api'

export default {
    name: 'ImportSE',
    data() {
        return {
            form: this.defaultForm(),
            query: this.defaultQuery(),
            appliedQuery: this.defaultQuery(),
            seList: [],
            importDialogVisible: false,
            reading: false,
            importing: false,
            loadingList: false,
            deletingSeId: ''
        }
    },
    computed: {
        filteredSeList() {
            return this.seList.filter(item => {
                const seid = String(item.se_id || '').toLowerCase()
                const cplc = String(item.cplc || '').toLowerCase()
                if (this.appliedQuery.seid && !seid.includes(this.appliedQuery.seid.toLowerCase())) return false
                if (this.appliedQuery.cplc && !cplc.includes(this.appliedQuery.cplc.toLowerCase())) return false
                if (this.appliedQuery.status && item.status !== this.appliedQuery.status) return false
                return true
            })
        }
    },
    created() {
        this.loadSecurityElements()
    },
    methods: {
        defaultForm() {
            return {
                seid: '',
                cplc: '',
                custodyLocation: ''
            }
        },

        defaultQuery() {
            return {
                seid: '',
                cplc: '',
                status: ''
            }
        },

        async loadSecurityElements() {
            this.loadingList = true
            try {
                const response = await serverSeApi.listSecurityElements()
                this.seList = response.data.data || []
            } catch (error) {
                this.$message.error(error.response?.data?.error || '查询 SE 失败')
            } finally {
                this.loadingList = false
            }
        },

        applyQuery() {
            this.appliedQuery = { ...this.query }
        },

        resetQuery() {
            this.query = this.defaultQuery()
            this.appliedQuery = this.defaultQuery()
        },

        openImportDialog() {
            this.form = this.defaultForm()
            this.importDialogVisible = true
        },

        async readCurrentSe() {
            this.reading = true
            try {
                const cplcResponse = await wailsSeApi.getCPLC()
                this.form.cplc = cplcResponse.data?.cplc_info || ''
                if (!this.form.cplc) {
                    throw new Error('未读取到 CPLC')
                }
                if (!this.form.seid.trim()) {
                    this.form.seid = this.suggestSeId()
                }
                this.$message.success('已读取当前 SE')
            } catch (error) {
                this.$message.error('读取 SE 失败: ' + error.message)
            } finally {
                this.reading = false
            }
        },

        async importCurrentSe() {
            if (!this.form.seid || !this.form.cplc) {
                this.$message.warning('请先读取当前 SE，并确认 SEID')
                return
            }
            this.importing = true
            try {
                await serverSeApi.createSecurityElement(this.form.seid.trim(), this.form.cplc, this.form.custodyLocation)
                this.$message.success('SE 已导入')
                this.importDialogVisible = false
                await this.loadSecurityElements()
            } catch (error) {
                this.$message.error(error.response?.data?.error || '导入 SE 失败')
            } finally {
                this.importing = false
            }
        },

        async deleteSe(row) {
            const seId = row?.se_id
            if (!seId) return
            try {
                await this.$confirm(`确认删除 SE ${seId} 的登记记录？`, '删除 SE', {
                    type: 'warning',
                    confirmButtonText: '删除',
                    cancelButtonText: '取消'
                })
            } catch {
                return
            }

            this.deletingSeId = seId
            try {
                await serverSeApi.deleteSecurityElement(seId)
                this.$message.success('SE 记录已删除')
                await this.loadSecurityElements()
            } catch (error) {
                this.$message.error(error.response?.data?.error || '删除 SE 失败')
            } finally {
                this.deletingSeId = ''
            }
        },

        suggestSeId() {
            const date = new Date()
            const yyyy = date.getFullYear()
            const mm = String(date.getMonth() + 1).padStart(2, '0')
            const dd = String(date.getDate()).padStart(2, '0')
            const count = this.seList.length + 1
            return `SE-${yyyy}${mm}${dd}-${String(count).padStart(3, '0')}`
        },

        formatTime(value) {
            return value ? new Date(value).toLocaleString() : '-'
        }
    }
}
</script>

<style scoped>
.header {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.query {
    margin-bottom: 12px;
}
</style>
