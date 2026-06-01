<template>
    <div class="page se-page">
        <div class="page-header">
            <div>
                <h2 class="page-title">安全芯片管理</h2>
                <p class="page-subtitle">安全芯片统一登记，不归属个人；私钥分片归属由持有人和安全芯片记录表达。</p>
            </div>
            <div>
                <el-button icon="el-icon-refresh" :loading="loadingList" @click="loadSecurityElements">刷新列表</el-button>
                <el-button type="primary" icon="el-icon-cpu" :loading="reading" @click="readCurrentSe">读取当前 SE</el-button>
            </div>
        </div>

        <el-card>
            <el-form :inline="true" :model="filters" class="filters">
                <el-form-item label="SEID">
                    <el-input v-model="filters.seid" clearable placeholder="按 SEID 筛选"></el-input>
                </el-form-item>
                <el-form-item label="CPLC">
                    <el-input v-model="filters.cplc" clearable placeholder="按 CPLC 筛选"></el-input>
                </el-form-item>
                <el-form-item label="状态">
                    <el-select v-model="filters.status" clearable placeholder="全部">
                        <el-option label="active" value="active"></el-option>
                        <el-option label="disabled" value="disabled"></el-option>
                        <el-option label="lost" value="lost"></el-option>
                        <el-option label="destroyed" value="destroyed"></el-option>
                    </el-select>
                </el-form-item>
            </el-form>

            <el-form :model="form" label-width="120px" class="import-form">
                <el-form-item label="安全芯片 ID">
                    <el-input v-model="form.seid" placeholder="系统自动建议，可按贴纸编号调整"></el-input>
                </el-form-item>
                <el-form-item label="CPLC">
                    <el-input v-model="form.cplc" type="textarea" :rows="3" readonly placeholder="点击读取当前 SE"></el-input>
                </el-form-item>
                <el-form-item label="保管位置">
                    <el-input v-model="form.custodyLocation" placeholder="例如 保险柜 A-01，可留空"></el-input>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" :loading="importing" @click="importCurrentSe">导入当前 SE</el-button>
                </el-form-item>
            </el-form>

            <el-table :data="filteredSeList" v-loading="loadingList" style="width: 100%">
                <el-table-column prop="se_id" label="SEID" width="170"></el-table-column>
                <el-table-column prop="cplc" label="CPLC" min-width="260"></el-table-column>
                <el-table-column prop="status" label="状态" width="100">
                    <template slot-scope="scope">
                        <el-tag :type="scope.row.status === 'active' ? 'success' : 'warning'">{{ scope.row.status }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="registered_by" label="登记人" width="130"></el-table-column>
                <el-table-column prop="remark" label="备注/位置" min-width="160"></el-table-column>
                <el-table-column prop="created_at" label="登记时间" width="170">
                    <template slot-scope="scope">
                        {{ formatTime(scope.row.created_at) }}
                    </template>
                </el-table-column>
                <el-table-column prop="last_used_at" label="最近使用时间" width="170">
                    <template slot-scope="scope">
                        {{ formatTime(scope.row.last_used_at) }}
                    </template>
                </el-table-column>
            </el-table>
        </el-card>
    </div>
</template>

<script>
import { seApi as wailsSeApi } from '../services/wails-api'
import { seApi as serverSeApi } from '../services/api'

export default {
    name: 'ImportSE',
    data() {
        return {
            form: {
                seid: '',
                cplc: '',
                custodyLocation: ''
            },
            filters: {
                seid: '',
                cplc: '',
                status: ''
            },
            seList: [],
            reading: false,
            importing: false,
            loadingList: false
        }
    },
    created() {
        this.loadSecurityElements()
    },
    computed: {
        filteredSeList() {
            return this.seList.filter(item => {
                if (this.filters.seid && !String(item.se_id || '').includes(this.filters.seid)) return false
                if (this.filters.cplc && !String(item.cplc || '').includes(this.filters.cplc)) return false
                if (this.filters.status && item.status !== this.filters.status) return false
                return true
            })
        }
    },
    methods: {
        async loadSecurityElements() {
            this.loadingList = true
            try {
                const response = await serverSeApi.listSecurityElements()
                this.seList = response.data.data || []
            } catch (error) {
                this.$message.error(error.response?.data?.error || '查询安全芯片失败')
            } finally {
                this.loadingList = false
            }
        },

        async readCurrentSe() {
            this.reading = true
            try {
                const cplcResponse = await wailsSeApi.getCPLC()
                this.form.cplc = cplcResponse.data?.cplc_info || ''
                if (!this.form.cplc) {
                    throw new Error('未读取到 CPLC')
                }
                this.form.seid = this.suggestSeId()
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
                this.$message.success('安全芯片已导入')
                this.form = { seid: '', cplc: '', custodyLocation: '' }
                await this.loadSecurityElements()
            } catch (error) {
                this.$message.error(error.response?.data?.error || '导入安全芯片失败')
            } finally {
                this.importing = false
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
.import-form {
    max-width: 720px;
    margin-bottom: 18px;
}

.filters {
    margin-bottom: 12px;
}
</style>
